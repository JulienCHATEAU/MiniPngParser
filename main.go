package main

import (
	"fmt"
	"os"
	"bytes"
	"strconv"
	"strings"
	"math"
	"io/ioutil"
	"encoding/binary"
)

/* Constants */

type ErrorCode byte 
const (
	Args_Count_Error 			= 1
	Read_File_Error				= 2
	Write_File_Error 			= 3
	File_Type_Error 			= 4
	Wrong_Block_Type_Error 		= 5
	Wrong_Pixel_Type_Error 		= 6
	Wrong_Comments_Length_Error	= 7
	Wrong_Data_Length_Error		= 8
	Wrong_Image_Dim_Error		= 9
	Too_Much_Data_Error			= 10
	Conversion_Error			= 11
)

type PixelType byte
const (
	Black_N_White 	= 0
	Grey_Scale 		= 1
	Palette 		= 2
	Color 			= 3
)

var BYTE_SIZE int = 8
var PGM string = "P2"
var PPM string = "P3"
var MP_TYPE []byte = []byte("Mini-PNG")
var HEADER byte = 72 // H
var COMMENTS byte = 67 // C
var DATA byte = 68 // D

/* Global Variables */

var dataPointer uint32
var fileContentLength uint32

/* MiniPng struct */

type MiniPng struct {
	filePath	string
	width		uint32
	height 		uint32
	pixelType 	PixelType
	comments	[]byte
	image		[]byte
}

func (mpFile *MiniPng) handleOneBlock(fileContent []byte) {
	switch (fileContent[dataPointer]) {
	case HEADER:
		mpFile.readHeader(fileContent)
		break;

	case COMMENTS:
		mpFile.readComments(fileContent)
		break;

	case DATA:
		mpFile.readImage(fileContent)
		break;

	default:
		logError("Can't read a block", Wrong_Block_Type_Error)
		break;
	}
}

func (mpFile *MiniPng) readHeader(fileContent []byte) {
	var headerContent []byte = []byte{}
	readTLVBlock(fileContent, &headerContent);

	fileWidth := headerContent[0:4]
	mpFile.width = binary.BigEndian.Uint32(fileWidth)

	fileHeight := headerContent[4:8]
	mpFile.height = binary.BigEndian.Uint32(fileHeight)

	filePixelType := headerContent[8]
	if filePixelType < 0 || filePixelType > 3 {
		logError("Can't read header", Wrong_Pixel_Type_Error)
	}
	mpFile.pixelType = PixelType(filePixelType)
}

func (mpFile *MiniPng) readComments(fileContent []byte) {
	readTLVBlock(fileContent, &mpFile.comments);
}

func (mpFile *MiniPng) readImage(fileContent []byte) {
	readTLVBlock(fileContent, &mpFile.image);
}

func (mpFile *MiniPng) printMetadata() {
	fmt.Println("Largeur :", mpFile.width)
	fmt.Println("Hauteur :", mpFile.height)
	fmt.Println("Type de pixel :", mpFile.pixelType)	
	fmt.Println("Commentaires : \"" + string(mpFile.comments) + "\"")	
}

func (mpFile *MiniPng) printImage() {
	switch (mpFile.pixelType) {
	case Black_N_White:
		// In terminal
		imageSize := mpFile.width * mpFile.height
		imageByteCapacity := math.Ceil(float64(imageSize) / float64(BYTE_SIZE))
		if float64(len(mpFile.image)) != imageByteCapacity {
			logError("Wrong image dimension", Wrong_Image_Dim_Error)		
		}
		flattenImage := make([]byte, imageSize)
		for pixelIndex, pixel := range mpFile.image {
			for i := 0; i < BYTE_SIZE; i++ { // Iterates pixel (1 byte) from right to left but should fill flattenImage from left to right
				pos := pixelIndex * BYTE_SIZE + BYTE_SIZE - 1 - i 
				if pos >= int(imageSize) {
					break;
				}
				flattenImage[pos] = getNthBit(pixel, i)
			}
		}
		for i := uint32(0); i < mpFile.height; i++ {
			for j := uint32(0); j < mpFile.width; j++ {
				if flattenImage[i * mpFile.width + j] == 0 {
					fmt.Print("X")
				} else {
					fmt.Print(" ")
				}
			}
			fmt.Print("\n")
		} 
		break;

	case Grey_Scale:
		// In PGM file
		pgmFilePath := strings.Replace(mpFile.filePath, ".mp", ".pgm", 1)
		pgmFileContent := []byte(mpFile.toPXMFormat(PGM))
		err := ioutil.WriteFile(pgmFilePath, pgmFileContent, 0644)
		if err != nil {
			logError("Can't write in file", Write_File_Error)
		}
		fmt.Println("You can run 'display " + pgmFilePath + "' to display the image")
		break;
	
	case Color:
		// In PPM file
		ppmFilePath := strings.Replace(mpFile.filePath, ".mp", ".ppm", 1)
		ppmFileContent := []byte(mpFile.toPXMFormat(PPM))
		err := ioutil.WriteFile(ppmFilePath, ppmFileContent, 0644)
		if err != nil {
			logError("Can't write in file", Write_File_Error)
		}
		fmt.Println("You can run 'display " + ppmFilePath + "' to display the image")
		break;

	default:
		logError("Pixel type is wrong", Wrong_Pixel_Type_Error)	
		break;	
	}
}

func (mpFile *MiniPng) toPXMFormat(format string) string {
	if mpFile.pixelType != Grey_Scale && format == PGM {
		logError("Can't convert non gray scale image to PGM format", Conversion_Error)	
	}
	if mpFile.pixelType != Color && format == PPM {
		logError("Can't convert non color image to PPM format", Conversion_Error)	
	}
	var pmgContent string = format + "\n"
	pmgContent += strconv.Itoa(int(mpFile.width)) + " " + strconv.Itoa(int(mpFile.height)) + "\n"
	pmgContent += "255\n"
	height := mpFile.height
	width := mpFile.width
	if format == PPM {
		height *= 3
	}
	for i := uint32(0); i < height; i++ {
		for j := uint32(0); j < width; j++ {
			pmgContent += strconv.Itoa(int(mpFile.image[i * width + j])) + " "
		}
		pmgContent += "\n"
	}
	return pmgContent
}

/* Util functions */

func readTLVBlock(fileContent []byte, contentDest *[]byte) {
	var contentStart uint32 = dataPointer + 5
	contentLength := binary.BigEndian.Uint32(fileContent[dataPointer+1:contentStart])
	var contentEnd uint32 = contentStart + contentLength
	if contentEnd > fileContentLength {
		logError("Block comments is too large", Wrong_Comments_Length_Error)
	}
	contentToAdd := fileContent[contentStart:contentEnd]
	*contentDest = append(*contentDest, contentToAdd...)
	dataPointer = contentEnd
}

func getNthBit(pixel byte, i int) byte {
	return ((pixel >> i) & 1);
}

func logError(errorMessage string, exitStatus int) {
	fmt.Println("[ERROR] " + errorMessage)
	os.Exit(exitStatus)
}

func checkFileType(fileContent []byte) {
	dataPointer = 8
	fileType := fileContent[:dataPointer]
	if !bytes.Equal(fileType, MP_TYPE) {
		logError("This file is not a Mini-PNG", File_Type_Error)
	}
}

/* Main */

func main() {
	argCount := len(os.Args)
	if argCount != 2 {
		logError("Command should be run with only one argument (the path of the file to parse).", Args_Count_Error)
	}

	filePath := os.Args[1]
	fileContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		logError("Can't read given file", Read_File_Error)
	}

	checkFileType(fileContent)
	var mpFile MiniPng = MiniPng{filePath, 0, 0, Black_N_White, []byte{}, []byte{}}
	fileContentLength = uint32(len(fileContent))
	for dataPointer < fileContentLength {
		mpFile.handleOneBlock(fileContent)
	}
	mpFile.printMetadata()
	mpFile.printImage()
}
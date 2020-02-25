package main

import (
	"fmt"
	"os"
	"bytes"
	"io/ioutil"
	"encoding/binary"
)

type ErrorCode byte 
const (
	Args_Count_Error 			= 1
	Read_File_Error				= 2
	File_Type_Error 			= 3
	Wrong_Block_Error 			= 4
	Wrong_Pixel_Type_Error 		= 5
	Wrong_Comments_Length_Error	= 6
	Too_Much_Data_Error			= 7
)

type PixelType byte
const (
	Black_N_White 	= 0
	Grey_Scale 		= 1
	Palette 		= 2
	Color 			= 3
)

var MP_TYPE []byte = []byte("Mini-PNG")
var HEADER byte = 72 // H
var COMMENTS byte = 67 // C
var DATA byte = 68 // C

var dataPointer uint32
var fileContentLength uint32

/* MiniPng struct */

type MiniPng struct {
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
		logError("Can't read a block", Wrong_Block_Error)
		break;
	}
}

func (mpFile *MiniPng) readHeader(fileContent []byte) {
	dataPointer = dataPointer + 1
	headerLength := binary.BigEndian.Uint32(fileContent[dataPointer:dataPointer + 4])
	dataPointer = dataPointer + 4
	if (dataPointer + headerLength > fileContentLength) {
		logError("contents length is to high", Wrong_Comments_Length_Error)
	}

	fileWidth := fileContent[dataPointer:dataPointer + 4]
	mpFile.width = binary.BigEndian.Uint32(fileWidth)
	dataPointer = dataPointer + 4

	fileHeight := fileContent[dataPointer:dataPointer + 4]
	mpFile.height = binary.BigEndian.Uint32(fileHeight)
	dataPointer = dataPointer + 4

	filePixelType := fileContent[dataPointer]
	if filePixelType < 0 || filePixelType > 3 {
		logError("Can't read header", Wrong_Pixel_Type_Error)
	}
	mpFile.pixelType = PixelType(filePixelType)
	dataPointer = dataPointer + 1
}

func (mpFile *MiniPng) readComments(fileContent []byte) {
	readBlock(fileContent, &mpFile.comments)
}

func (mpFile *MiniPng) readImage(fileContent []byte) {
	readBlock(fileContent, &mpFile.image)
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
		imageSize := mpFile.width * mpFile.height
		flattenImage := make([]byte, imageSize)
		for pixelIndex, pixel := range mpFile.image {
			for i := 0; i < 8; i++ {
				pos := pixelIndex * 8 + 7 - i
				if pos >= int(imageSize) {
					break;
				}
				flattenImage[pos] = getNthBit(pixel, i)
			}
		}
		for height := uint32(0); height < mpFile.height; height++ {
			for width := uint32(0); width < mpFile.width; width++ {
				if flattenImage[height * mpFile.width + width] == 0 {
					fmt.Print("X")
				} else {
					fmt.Print(" ")
				}
			}
			fmt.Print("\n")
		} 
		break;

	default:
		logError("Pixel type is wrong", Wrong_Pixel_Type_Error)	
		break;	
	}
}

/* Core functions */

func getNthBit(pixel byte, i int) byte {
	return ((pixel >> i) & 1);
}

func readBlock(fileContent []byte, contentDest *[]byte) {
	var contentStart uint32 = dataPointer + 5
	contentLength := binary.BigEndian.Uint32(fileContent[dataPointer+1:contentStart])
	var contentEnd uint32 = contentStart + contentLength
	if (contentEnd > fileContentLength) {
		logError("contents length is to high", Wrong_Comments_Length_Error)
	}
	*contentDest = make([]byte, contentLength)
	copy(*contentDest, fileContent[contentStart:contentEnd])
	dataPointer = contentEnd
}

func logError(errorMessage string, exitStatus int) {
	fmt.Println("[ERROR] " + errorMessage)
	os.Exit(exitStatus)
}

func checkFileType(fileContent []byte) {
	fileType := fileContent[:8]
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
	for _, b := range fileContent {
		fmt.Printf("%x | ", b);
	}
	fmt.Println()
	if err != nil {
		logError("Can't read given file", Read_File_Error)
	}

	checkFileType(fileContent)
	var mpFile MiniPng = MiniPng{0, 0, Black_N_White, []byte{}, []byte{}}
	dataPointer = 8
	fileContentLength = uint32(len(fileContent))
	for dataPointer < fileContentLength {
		mpFile.handleOneBlock(fileContent)
	}
	if dataPointer < fileContentLength {
		logError("Too much data in this image", Too_Much_Data_Error)
	}
	mpFile.printMetadata()
	mpFile.printImage()
}
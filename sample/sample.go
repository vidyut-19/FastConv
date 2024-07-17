package main

import (
	"os"
	"proj3/png"
)

func main() {

	//Assumes the user specifies a file as the first argument
	filePath := os.Args[1]

	//Loads the png image and returns the image or an error
	pngImg, err := png.Load(filePath)

	if err != nil {
		panic(err)
	}

	//Performs a grayscale filtering effect on the image
	pngImg.Grayscale(0, 0)

	//Saves the image to a new file
	_ = pngImg.Save("test_gray.png")

	//Checks to see if there were any errors when saving.
	if err != nil {
		panic(err)
	}

}

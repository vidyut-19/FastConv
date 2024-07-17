package scheduler

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"io"
	"log"
	"os"
	"proj3/png"
	myPng "proj3/png"
	"strings"
)

func RunSequential(config Config) {
	// Example of initializing and using the Effects struct
	sizesString := config.DataDirs
	sizes := strings.Split(sizesString, "+")
	for _, size := range sizes {
		dirPath := "../data/in/"
		images := getJSON(false)
		for _, task := range images {
			task.Size = size
			task.processImage(dirPath)
		}
	}

}

// struct to wrap the attributes of each image we wish to process as a task
type ImageTask struct {
	InPath     string     `json:"inPath"`
	OutPath    string     `json:"outPath"`
	Effects    []string   `json:"effects"`
	Size       string     //get the size of image from CLI
	Image      *png.Image //pointer to the image object for splitting
	ChunkStart int        // starting y-coordinate of chunk
	Top        bool       // indicates if the chunk is the top chunk
	Bottom     bool       // indicates if the chunk is the bottom chunk
	ChunkEnd   int        // ending y-coordinate of chunk
}

func ApplyEffects(task *ImageTask, par bool, startY int, endY int) *ImageTask {
	e := png.NewEffects()
	for i, effect := range task.Effects {
		if i > 0 {
			task.Image.In = task.Image.Out
			task.Image.Out = image.NewRGBA64(task.Image.In.Bounds())
		}
		switch effect {
		case "S": // Sharpen
			task.Image.ApplyEffect(e.S, par, startY, endY)
		case "E": // Edge Detection
			task.Image.ApplyEffect(e.E, par, startY, endY)
		case "B": // Blurx
			task.Image.ApplyEffect(e.B, par, startY, endY)
		case "G": // Grayscale
			task.Image.Grayscale(0, 0)
		default:
			continue
		}
	}
	return task
}

// this function actually processes each image (used in parfiles as well)
func (task ImageTask) processImage(dirPath string) {
	location := dirPath + "/" + task.Size + "/" + task.InPath
	img, err := myPng.Load(location)
	if err != nil {
		panic(err)
	}
	img.ApplyEffects(task.Effects, false, 0, 0)
	task.OutPath = task.Size + "_" + task.OutPath
	outPath := "../data/out/" + task.OutPath
	_ = img.Save(outPath)

	if err != nil {
		panic(err)
	}
}

func (task ImageTask) ProcessSlice() {

	e := png.NewEffects()
	var temp *image.RGBA64
	for i, effect := range task.Effects {
		if i > 0 {
			task.Image.Out = temp
		}
		switch effect {
		case "S": // Sharpen
			task.Image.ApplyEffect(e.S, true, task.ChunkStart, task.ChunkEnd)
		case "E": // Edge Detection
			task.Image.ApplyEffect(e.E, true, task.ChunkStart, task.ChunkEnd)
		case "B": // Blurx
			task.Image.ApplyEffect(e.B, true, task.ChunkStart, task.ChunkEnd)
		case "G": // Grayscale
			task.Image.Grayscale(task.ChunkStart, task.ChunkEnd)
		}
		temp = task.Image.In
		task.Image.In = task.Image.Out
	}

}

// this function reads the effects file and returns details about the images to process
func getJSON(toChunk bool) []*ImageTask {
	effectsPathFile := fmt.Sprintf("../data/effects.txt")
	effectsFile, _ := os.Open(effectsPathFile)
	reader := json.NewDecoder(effectsFile)

	var images []*ImageTask
	//toChunk will be false if we are running the sequential scheduler

	for {
		var imageTask *ImageTask
		if err := reader.Decode(&imageTask); err != nil {
			if err == io.EOF {
				break // End of file reached
			}
			log.Fatalf("error decoding JSON: %v", err)
		}
		if !toChunk {
			images = append(images, imageTask)
		} else {

		}
	}
	return images

}

func AddChunk(masterImage *png.Image, chunk *ImageTask) {
	bounds := chunk.Image.Out.Bounds()
	var startY int
	//padding correction
	if chunk.Top {
		bounds.Max.Y -= 2
	} else if chunk.Bottom {
		bounds.Min.Y += 2
		startY += 2
	} else {
		bounds.Min.Y += 2
		startY += 2
		bounds.Max.Y -= 2
	}
	//cut out the overlapping padding that we added earlier when splitting
	imgCrop := chunk.Image.Out.SubImage(image.Rect(0, startY, bounds.Max.X, bounds.Max.Y))
	draw.Draw(masterImage.Out, bounds, imgCrop, image.Point{0, startY}, draw.Src)

}

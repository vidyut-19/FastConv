package scheduler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	myPng "proj3/png"
	"runtime"
	"strings"
)

func RunPipeline(config Config) {
	dirPath := "../data/in/"
	runtime.GOMAXPROCS(config.ThreadCount)
	generator := func(done <-chan interface{}, config Config) <-chan *ImageTask {
		sizesString := config.DataDirs
		sizes := strings.Split(sizesString, "+")
		taskStream := make(chan *ImageTask)
		effectsPathFile := fmt.Sprintf("../data/effects.txt")
		effectsFile, _ := os.Open(effectsPathFile)
		reader := json.NewDecoder(effectsFile)
		// we spawn goroutines for images of all sizes

		go func() {
			defer close(taskStream)
			for {
				var imageTask *ImageTask
				if err := reader.Decode(&imageTask); err != nil {
					if err == io.EOF {
						break
					}
					log.Fatalf("error decoding JSON: %v", err)
				}
				for _, size := range sizes {
					imageTask.Size = size
					location := dirPath + imageTask.Size + "/" + imageTask.InPath
					img, err := myPng.Load(location)
					imageTask.Image = img
					if err != nil {
						panic(err)
					}
					select {
					case <-done:
						return
					case taskStream <- imageTask:
					}
				}
			}
		}()

		return taskStream
	}

	processor := func(
		done <-chan interface{},
		taskStream <-chan *ImageTask,
	) <-chan *ImageTask {
		imageStream := make(chan *ImageTask)
		go func() {
			defer close(imageStream)
			for i := range taskStream {
				select {
				case <-done:
					return
				case imageStream <- ApplyEffects(i, false, 0, 0):
				}
			}
		}()
		return imageStream
	}
	collector := func(
		done <-chan interface{},
		imageStream <-chan *ImageTask,
	) <-chan bool {
		completed := make(chan bool)
		go func() {
			defer close(completed)
			for task := range imageStream {
				task.OutPath = task.Size + "_" + task.OutPath
				outPath := "../data/out/" + task.OutPath
				select {
				case <-done:
					return
				case completed <- task.Image.Save(outPath):
				}
			}
		}()
		return completed
	}
	done := make(chan interface{})
	defer close(done)

	pipeline := collector(done, processor(done, generator(done, config)))
	for _ = range pipeline {
		continue
	}
}

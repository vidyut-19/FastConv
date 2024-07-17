package scheduler

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"proj3/png"
	"strings"
	"sync"
)

func RunDeque(config Config) {
	numThreads := config.ThreadCount
	//sends a stream of chunked image tasks
	//receives a stream of chunked image tasks
	//indicates that the image has been saved

	//initiate numThreads number of work pools
	var wps []WorkPool
	for i := 0; i < numThreads; i++ {
		wps = append(wps, NewQueue())
	}

	dirPath := "../data/in/"
	sizesString := config.DataDirs
	sizes := strings.Split(sizesString, "+")
	MainImageTasks := make(chan *ImageTask)
	effectsPathFile := "../data/effects.txt"
	effectsFile, _ := os.Open(effectsPathFile)
	reader := json.NewDecoder(effectsFile)
	//Generator stage, add tasks to the MainImageTasks channel
	go func() {
		defer close(MainImageTasks)
		for {
			var MainTask *ImageTask
			if err := reader.Decode(&MainTask); err != nil {
				if err == io.EOF {
					break // End of file reached
				}
				log.Fatalf("error decoding JSON: %v", err)
			}
			for _, size := range sizes {

				MainTask.Size = size

				location := dirPath + "/" + MainTask.Size + "/" + MainTask.InPath
				pngImage, err := png.Load(location)
				if err != nil {
					panic(err)
				}
				MainTask.Image = pngImage
				MainImageTasks <- MainTask

			}
		}

	}()

	i := 0
	for task := range MainImageTasks {
		wps[i].PushBottom(task)
		i = (i + 1) % numThreads
	}
	var taskOut []*ImageTask
	var allTasks [][]*ImageTask
	var wg sync.WaitGroup
	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func(taskOut []*ImageTask) {
			localPool := Worker(i, wps, &wg)
			allTasks = append(allTasks, localPool)
		}(taskOut)
		wg.Wait()
	}
	for _, tasks := range allTasks {
		taskOut = append(taskOut, tasks...)

	}
	for _, task := range taskOut {
		task.OutPath = task.Size + "_" + task.OutPath
		outPath := "../data/out/" + task.OutPath
		_ = task.Image.Save(outPath)
	}

}

func Worker(id int, wps []WorkPool, wg *sync.WaitGroup) []*ImageTask {
	var localPool []*ImageTask
	wp := wps[id] //identify our work pool
	for {
		task := wp.PopBottom() // Implement PopFront to get a task from the front
		if task == nil {
			success := wp.Steal(id, wps) // Implement Steal to try to steal a task from the back
			if !success {
				wg.Done()
				return localPool
			}
			task = wp.PopBottom() // Retrieve the stolen task
		}
		task = ApplyEffects(task, false, 0, 0)
		localPool = append(localPool, task)
	}
}

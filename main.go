package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path"
	"strings"

	codec "github.com/alacrity-engine/resource-codec"
	"github.com/golang-collections/collections/queue"
	bolt "go.etcd.io/bbolt"
)

const bucketName = "pictures"

var (
	projectPath      string
	resourceFilePath string
)

func parseFlags() {
	flag.StringVar(&projectPath, "project", ".",
		"Path to the project to pack spritesheets for.")
	flag.StringVar(&resourceFilePath, "out", "./stage.res",
		"Resource file to store animations and pictures.")

	flag.Parse()
}

func loadPicture(pic string) (*codec.PictureData, error) {
	file, err := os.Open(pic)

	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)

	if err != nil {
		return nil, err
	}

	return codec.NewPictureFromImage(img)
}

func main() {
	parseFlags()

	// Open the resource file.
	resourceFile, err := bolt.Open(resourceFilePath, 0666, nil)
	handleError(err)
	defer resourceFile.Close()

	// Create collections for pictures.
	err = resourceFile.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte(bucketName))

		if err != nil {
			return err
		}

		return nil
	})
	handleError(err)

	entries, err := os.ReadDir(projectPath)
	handleError(err)

	traverseQueue := queue.New()

	if len(entries) <= 0 {
		return
	}

	for _, entry := range entries {
		traverseQueue.Enqueue(FileTracker{
			EntryPath: ".",
			Entry:     entry,
		})
	}

	for traverseQueue.Len() > 0 {
		fsEntry := traverseQueue.Dequeue().(FileTracker)

		if fsEntry.Entry.IsDir() {
			entries, err = os.ReadDir(path.Join(fsEntry.EntryPath, fsEntry.Entry.Name()))
			handleError(err)

			for _, entry := range entries {
				traverseQueue.Enqueue(FileTracker{
					EntryPath: path.Join(fsEntry.EntryPath, fsEntry.Entry.Name()),
					Entry:     entry,
				})
			}

			continue
		}

		if !strings.HasSuffix(fsEntry.Entry.Name(), ".anim.yml") {
			continue
		}

		// Load the picture.
		pic, err := loadPicture(path.Join(fsEntry.EntryPath, fsEntry.Entry.Name()))
		handleError(err)

		// Compress the picture.
		compressedPicture, err := pic.Compress()
		handleError(err)

		// Serialize picture data to byte array.
		pictureBytes, err := compressedPicture.ToBytes()
		handleError(err)

		// Save the picture to the database.
		pictureID := strings.TrimSuffix(path.Base(fsEntry.Entry.Name()),
			path.Ext(fsEntry.Entry.Name()))
		err = resourceFile.Update(func(tx *bolt.Tx) error {
			buck := tx.Bucket([]byte(bucketName))

			if buck == nil {
				return fmt.Errorf("no pictures bucket present")
			}

			err = buck.Put([]byte(pictureID), pictureBytes)

			if err != nil {
				return err
			}

			return nil
		})
		handleError(err)
	}
}

func handleError(err error) {
	if err != nil {
		panic(err)
	}
}

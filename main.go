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
	bolt "go.etcd.io/bbolt"
)

const bucketName = "pictures"

var (
	picturesPath     string
	resourceFilePath string
)

func parseFlags() {
	flag.StringVar(&picturesPath, "pictures", "./pictures",
		"Path to the directory where pictures are stored.")
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

	// Get pictures from the directory.
	pictures, err := os.ReadDir(picturesPath)
	handleError(err)
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

	for _, pictureInfo := range pictures {
		if pictureInfo.IsDir() {
			fmt.Println("Error: directory found in the spritesheets folder.")
			os.Exit(1)
		}

		// Load the picture.
		pic, err := loadPicture(path.Join(picturesPath,
			pictureInfo.Name()))
		handleError(err)

		// Compress the picture.
		compressedPicture, err := pic.Compress()
		handleError(err)

		// Serialize picture data to byte array.
		pictureBytes, err := compressedPicture.ToBytes()
		handleError(err)

		// Save the picture to the database.
		pictureID := strings.TrimSuffix(path.Base(pictureInfo.Name()),
			path.Ext(pictureInfo.Name()))
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

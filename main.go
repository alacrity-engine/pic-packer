package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"os"
	"path"
	"strings"

	codec "github.com/alacrity-engine/resource-codec"
	"github.com/boltdb/bolt"
	"github.com/faiface/pixel"
)

var (
	spritesheetsPath string
	resourceFilePath string
)

func parseFlags() {
	flag.StringVar(&spritesheetsPath, "spritesheets", "./spritesheets",
		"Path to the directory where spritesheets are stored.")
	flag.StringVar(&resourceFilePath, "out", "./stage.res",
		"Resource file to store animations and spritesheets.")

	flag.Parse()
}

func loadPicture(pic string) (*pixel.PictureData, error) {
	file, err := os.Open(pic)

	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)

	if err != nil {
		return nil, err
	}

	return pixel.PictureDataFromImage(img), nil
}

func main() {
	parseFlags()

	// Get spritesheets from the directory.
	spritesheets, err := ioutil.ReadDir(spritesheetsPath)
	handleError(err)
	// Open the resource file.
	resourceFile, err := bolt.Open(resourceFilePath, 0666, nil)
	handleError(err)
	defer resourceFile.Close()

	// Create collections for spritesheets and animations.
	err = resourceFile.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte("spritesheets"))

		if err != nil {
			return err
		}

		return nil
	})
	handleError(err)

	for _, spritesheetInfo := range spritesheets {
		if spritesheetInfo.IsDir() {
			fmt.Println("Error: directory found in the spritesheets folder.")
			os.Exit(1)
		}

		// Load the spritesheet picture.
		spritesheet, err := loadPicture(path.Join(spritesheetsPath,
			spritesheetInfo.Name()))
		handleError(err)

		// Serialize picture data to byte array.
		spritesheetBytes, err := codec.PictureDataToBytes(spritesheet)
		handleError(err)

		// Save the spritesheet to the database.
		spritesheetID := strings.TrimSuffix(path.Base(spritesheetInfo.Name()),
			path.Ext(spritesheetInfo.Name()))
		err = resourceFile.Update(func(tx *bolt.Tx) error {
			buck := tx.Bucket([]byte("spritesheets"))

			if buck == nil {
				return fmt.Errorf("no spritesheets bucket present")
			}

			err = buck.Put([]byte(spritesheetID), spritesheetBytes)

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

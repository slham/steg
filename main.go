package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"

	"github.com/auyer/steganography"
	"github.com/sirupsen/logrus"
)

const (
	hiddenBitMask = 1
)

var validExtensions = []string{"jpg", "png"}

func main() {
	var verbose bool
	var encode bool
	var decode bool
	var imagePath string
	var secret string
	var secretPath string

	flag.BoolVar(&verbose, "verbose", false, "verbose logging")
	flag.BoolVar(&encode, "encode", false, "encode image file")
	flag.BoolVar(&decode, "decode", false, "decode image file")
	flag.StringVar(&imagePath, "image-path", "", "path to image")
	flag.StringVar(&secret, "secret", "", "secret message")
	flag.StringVar(&secretPath, "secret-path", "", "path to secret file")

	flag.Parse()

	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if !encode && !decode || encode && decode {
		logrus.Warnf("must pass either -encode or -decode")
		logrus.Infof("exiting")
		return
	}

	if imagePath == "" {
		logrus.Warnf("must pass -image-path")
		logrus.Infof("exiting")
		return
	}

	img, err := openImage(imagePath)
	if err != nil {
		logrus.WithError(err).Errorf("failed to decode image")
		logrus.Infof("exiting")
		return
	}

	if decode {
		sizeOfMessage := steganography.GetMessageSizeFromImage(img)
		message := steganography.Decode(sizeOfMessage, img)
		logrus.Infof("Steganography completed successfully!")
		logrus.Infof("Hidden messaage: %s", message)
		return
	}

	if encode {
		if secret == "" && secretPath == "" || secret != "" && secretPath != "" {
			logrus.Warnf("must pass -secret or -secret-path")
			logrus.Infof("exiting")
			return
		}

		var message string
		logrus.Debugf("getting secret")

		if secret != "" {
			message = secret
		} else {
			// Open the secret message file
			secretFile, err := os.Open(secretPath)
			if err != nil {
				logrus.WithError(err).Errorf("could not open secret (%s)", secretPath)
				logrus.Infof("exiting")
				return
			}
			defer secretFile.Close()

			// Read the secret message from file
			message = readSecretMessage(secretFile)
		}

		encodedImg := new(bytes.Buffer)
		err = steganography.Encode(encodedImg, img, []byte(message))
		if err != nil {
			log.Fatalf("Error encoding message into file  %v", err)
		}
		outFile, err := os.Create("encoded_image.jpg")
		if err != nil {
			log.Fatalf("Error creating file: %v", err)
		}
		bufio.NewWriter(outFile).Write(encodedImg.Bytes())

		logrus.Infof("Steganography completed successfully!")
		logrus.Infof("file can be found at encoded_image.jpg")
		return
	}

	return
}

//Encode image file
func encodeImage(name string, extension string, img image.Image) error {
	logrus.Debugf("encoding image")
	path := fmt.Sprintf("%s.%s", name, extension)

	// Save the encoded image to a file
	encodedFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("%w. failed to create file (%s)", err, path)
	}
	defer encodedFile.Close()

	// Encode the new image and save it
	switch extension {
	case "jpg":
		err = jpeg.Encode(encodedFile, img, nil)
		if err != nil {
			return fmt.Errorf("%w. failed to encode file", err)
		}
	case "png":
		err = png.Encode(encodedFile, img)
		if err != nil {
			return fmt.Errorf("%w. failed to encode file", err)
		}
	}

	return nil
}

//Decode image file
func openImage(filename string) (image.Image, error) {
	logrus.Debugf("opening image")
	inFile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer inFile.Close()
	reader := bufio.NewReader(inFile)
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// Reads the secret message from a file
func readSecretMessage(file *os.File) string {
	// Read the file contents
	messageBytes := make([]byte, 0)
	buffer := make([]byte, 1024)
	for {
		n, err := file.Read(buffer)
		if err != nil {
			break
		}
		messageBytes = append(messageBytes, buffer[:n]...)
	}

	// Convert the message bytes to a string
	message := string(messageBytes)

	return message
}

// Checks if the secret message can fit within the image
func canFitMessage(img image.Image, message string) bool {
	maxMessageSize := img.Bounds().Max.X * img.Bounds().Max.Y * 3 / 8
	return len(message) < maxMessageSize
}

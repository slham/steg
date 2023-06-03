package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"strings"

	"github.com/samber/lo"
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

	parts := strings.Split(imagePath, ".")
	if len(parts) != 2 {
		logrus.Errorf("invalid image-path (%s). example path: 'image.jpg'", imagePath)
		logrus.Infof("exiting")
		return
	}

	imageName := parts[0]
	imageExtension := parts[1]
	if !lo.Contains(validExtensions, imageExtension) {
		logrus.Errorf("invalid image-path (%s). valid file types are (%v)", imagePath, validExtensions)
		logrus.Infof("exiting")
		return
	}

	img, err := decodeImage(imageName, imageExtension)
	if err != nil {
		logrus.WithError(err).Errorf("failed to decode image")
		logrus.Infof("exiting")
		return
	}

	if decode {
		// Decode the secret message from the image
		message := decodeSecretMessage(img)
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

		// Check if the message can fit within the image
		if !canFitMessage(img, message) {
			logrus.Warnf("Secret message file is too large to fit within the image")
			logrus.Infof("exiting")
			return
		}

		// Create a new image for the steganography result
		encodedImg := image.NewRGBA(img.Bounds())

		// Embed the secret message into the new image
		embedSecretMessage(img, encodedImg, message)

		if err := encodeImage("encoded_image", imageExtension, img); err != nil {
			logrus.WithError(err).Errorf("failed to encode image")
			logrus.Infof("exiting")
			return
		}

		logrus.Infof("Steganography completed successfully!")
		logrus.Infof("file can be found at encoded_image.%s", imageExtension)
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
func decodeImage(name string, extension string) (image.Image, error) {
	logrus.Debugf("decoding image")
	// Open the image file
	var img image.Image
	var err error
	imageFilePath := fmt.Sprintf("%s.%s", name, extension)
	imageFile, err := os.Open(imageFilePath)
	if err != nil {
		return img, fmt.Errorf("%w. failed to open image file from path (%s)", err, imageFilePath)
	}
	defer imageFile.Close()

	// Decode the image
	switch extension {
	case "jpg":
		img, err = jpeg.Decode(imageFile)
		if err != nil {
			return img, fmt.Errorf("%w. failed to decode file", err)
		}
	case "png":
		img, err = png.Decode(imageFile)
		if err != nil {
			return img, fmt.Errorf("%w. failed to decode file", err)
		}
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

// Embeds the secret message into the image using LSB steganography
func embedSecretMessage(originalImg image.Image, encodedImg *image.RGBA, message string) {
	logrus.Debugf("embedding secret")
	bounds := originalImg.Bounds()

	// Iterate over each pixel in the image
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Get the original pixel color
			originalColor := originalImg.At(x, y)

			// Convert the color to RGBA
			r, g, b, a := originalColor.RGBA()
			originalRGBAColor := color.RGBA{
				uint8(r >> 8),
				uint8(g >> 8),
				uint8(b >> 8),
				uint8(a >> 8),
			}

			// Get the next bit of the secret message
			bit := getNextMessageBit(message)

			// Modify the least significant bit of the red channel
			encodedRGBAColor := originalRGBAColor
			encodedRGBAColor.R = (originalRGBAColor.R &^ hiddenBitMask) | (bit & hiddenBitMask)

			// Set the modified pixel color in the new image
			encodedImg.Set(x, y, encodedRGBAColor)
		}
	}
}

// Gets the next bit of the secret message
func getNextMessageBit(message string) uint8 {
	if len(message) > 0 {
		// Get the first character of the message
		char := message[0]

		// Remove the first character from the message
		message = message[1:]

		// Convert the character to its ASCII representation
		ascii := uint8(char)

		// Return the least significant bit of the ASCII value
		return ascii & hiddenBitMask
	}

	// Return 0 if the message is empty
	return 0
}

// Decodes the secret message from the image using LSB steganography
func decodeSecretMessage(encodedImg image.Image) string {
	bounds := encodedImg.Bounds()
	messageBytes := make([]byte, 0)

	// Iterate over each pixel in the image
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Get the pixel color
			encodedColor := encodedImg.At(x, y)

			// Convert the color to RGBA
			r, _, _, _ := encodedColor.RGBA()
			encodedRGBAColor := color.RGBA{
				uint8(r >> 8),
				0,
				0,
				0,
			}

			// Get the least significant bit of the red channel
			bit := encodedRGBAColor.R & hiddenBitMask

			// Append the bit to the message bytes
			messageBytes = append(messageBytes, bit)

			// Check if the message termination character is encountered
			if len(messageBytes) >= 8 && bytes.Equal(messageBytes[len(messageBytes)-8:], []byte{0, 0, 0, 0, 0, 0, 0, 0}) {
				break
			}
		}

		logrus.Debugf("messageBytes: %+v", messageBytes)
		logrus.Debugf("messageBytes: %s", string(messageBytes))

		// Check if the message termination character is encountered
		if len(messageBytes) >= 8 && bytes.Equal(messageBytes[len(messageBytes)-8:], []byte{0, 0, 0, 0, 0, 0, 0, 0}) {
			break
		}
	}

	// Remove the message termination character from the message bytes
	messageBytes = messageBytes[:len(messageBytes)-8]

	// Convert the message bytes to a string
	message := string(messageBytes)

	return message
}

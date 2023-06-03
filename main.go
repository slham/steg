package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"os"
)

const (
	hiddenBitMask = 1
)

func main() {
	// Open the original image file
	originalFile, err := os.Open("original_image.jpg")
	if err != nil {
		log.Fatal(err)
	}
	defer originalFile.Close()

	// Decode the original image
	originalImg, err := jpeg.Decode(originalFile)
	if err != nil {
		log.Fatal(err)
	}

	// Open the secret message file
	secretFile, err := os.Open("secret.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer secretFile.Close()

	// Read the secret message from file
	message := readSecretMessage(secretFile)

	// Check if the message can fit within the image
	if !canFitMessage(originalImg, message) {
		log.Fatal("Secret message is too large to fit within the image")
	}

	// Create a new image for the steganography result
	encodedImg := image.NewRGBA(originalImg.Bounds())

	// Embed the secret message into the new image
	embedSecretMessage(originalImg, encodedImg, message)

	// Save the encoded image to a file
	encodedFile, err := os.Create("encoded_image.jpg")
	if err != nil {
		log.Fatal(err)
	}
	defer encodedFile.Close()

	// Encode the new image and save it
	err = jpeg.Encode(encodedFile, encodedImg, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Steganography completed successfully!")
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

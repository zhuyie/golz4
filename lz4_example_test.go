package golz4

import (
	"bytes"
	crypto_rand "crypto/rand"
	"fmt"
	"math/rand"
	"sync"
)

func ExampleCompress() {
	// Generate some random bytes
	plaintext := make([]byte, 200)
	_, err := crypto_rand.Read(plaintext)
	if err != nil {
		fmt.Println("rand.Read error: ", err)
		return
	}

	// Compress
	compressed := make([]byte, CompressBound(len(plaintext)))
	n, err := Compress(compressed, plaintext)
	if err != nil {
		fmt.Println("Compress error: ", err)
		return
	}
	compressed = compressed[:n]

	// Decompress
	decompressed := make([]byte, len(plaintext))
	n, err = Decompress(decompressed, compressed)
	if err != nil {
		fmt.Println("Decompress error: ", err)
		return
	}
	decompressed = decompressed[:n]

	// Check
	if !bytes.Equal(plaintext, decompressed) {
		fmt.Println("Not equal")
		return
	}
	fmt.Println("Equal")

	// Output: Equal
}

func ExampleContinueCompress() {
	netChannel := make(chan []byte, 10)
	var wg sync.WaitGroup
	wg.Add(2)

	// The sender
	go func() {
		defer wg.Done()
		cc := NewContinueCompress(4096, 256)
		for i := 0; i < 5; i++ {
			// a net packet...
			packet := make([]byte, 64+rand.Intn(64))
			_, err := crypto_rand.Read(packet)
			if err != nil {
				fmt.Println("rand.Read error: ", err)
				break
			}

			// Compress the packet
			err = cc.Write(packet)
			if err != nil {
				fmt.Println("cc.Write error: ", err)
				break
			}
			compressed := make([]byte, CompressBound(cc.MsgLen()))
			n, err := cc.Process(compressed)
			if err != nil {
				fmt.Println("cc.Process error: ", err)
				break
			}
			compressed = compressed[:n]

			// send to the other side
			netChannel <- packet
			netChannel <- compressed
		}
		close(netChannel)
	}()

	// The receiver
	go func() {
		defer wg.Done()
		cd := NewContinueDecompress(4096, 256)
		for {
			plaintext, ok := <-netChannel
			if !ok {
				break
			}
			compressed := <-netChannel

			// Decompress the packet
			decompressed := make([]byte, cd.MaxMessageSize())
			n, err := cd.Process(decompressed, compressed)
			if err != nil {
				fmt.Println("cd.Process error: ", err)
				break
			}
			decompressed = decompressed[:n]

			// Check
			if !bytes.Equal(plaintext, decompressed) {
				fmt.Println("Not equal")
				break
			}
			fmt.Println("Equal")
		}
	}()

	wg.Wait()

	// Output:
	// Equal
	// Equal
	// Equal
	// Equal
	// Equal
}

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
	plaintext1 := make([]byte, 100)
	_, err := crypto_rand.Read(plaintext1)
	if err != nil {
		fmt.Println("rand.Read error: ", err)
		return
	}
	plaintext2 := make([]byte, 200)
	_, err = crypto_rand.Read(plaintext2)
	if err != nil {
		fmt.Println("rand.Read error: ", err)
		return
	}

	// pass nil as dst will allocate a new one
	compressed, err := Compress(nil, plaintext1)
	if err != nil {
		fmt.Println("Compress error: ", err)
		return
	}
	compressedLen1 := len(compressed)

	// append compressed plaintext2 to compressed
	compressed, err = Compress(compressed, plaintext2)
	if err != nil {
		fmt.Println("Compress error: ", err)
		return
	}

	// dst must be preallocated and have enough capacity
	decompressed1 := make([]byte, 0, 100)
	decompressed1, err = Decompress(decompressed1, compressed[:compressedLen1])
	if err != nil {
		fmt.Println("Decompress error: ", err)
		return
	}
	decompressed2 := make([]byte, 0, 200)
	decompressed2, err = Decompress(decompressed2, compressed[compressedLen1:])
	if err != nil {
		fmt.Println("Decompress error: ", err)
		return
	}

	// check
	if !bytes.Equal(plaintext1, decompressed1) || !bytes.Equal(plaintext2, decompressed2) {
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
			compressed, err := cc.Process(nil)
			if err != nil {
				fmt.Println("cc.Process error: ", err)
				break
			}

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
			decompressed, err := cd.Process(compressed, nil)
			if err != nil {
				fmt.Println("cd.Process error: ", err)
				break
			}
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

package gok

import (
	"fmt"
	"testing"
	"time"
)

func Test_Gok(t *testing.T) {
	// init a Gok
	gok := NewGok(100,10)

	// Add a Gokit
	errBus := gok.AddGokitFunc(func() error {
		fmt.Println(1)
		return nil
	},"print","print-1")

	// Read From ErrorBus
	go func() {
		for {
			err := errBus.Next()
			t.Log(err)
		}
	}()

	// Start Gok
	gok.Run()

	// Add another Gokit
	_ = gok.AddGokitFunc(func() error {
		fmt.Println(2)
		return nil
	},"print","print-2")

	time.Sleep(100 * time.Millisecond)
	// Pause Gok
	gok.Pause()

	// Add other Gokits
	_ = gok.AddGokitFunc(func() error {
		fmt.Println(3)
		return nil
	},"print","print-3")
	_ = gok.AddGokitFunc(func() error {
		fmt.Println(4)
		return nil
	},"print","print-4")

	// Restart Gok
	gok.Restart()

	// Waiting Gokit Execute ...
	time.Sleep(time.Hour)
}


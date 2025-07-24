package api

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

func HelloHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	result := 0
	for i := 0; i < 100000; i++ {
		result += i * 2
	}
	fmt.Println("Tính toán tốn thời gian:", result)

	fmt.Fprintf(w, "Test init")
}

go run cmd/main.go to run server

wrk -t 4 -c 40 -d 5s http://localhost:8081/init to test api rate limit
"-t: thresh of device"
"-c total of request"
"-d time process"

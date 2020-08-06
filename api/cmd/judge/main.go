package main

import (
	"judge-two/internal/api"
)

func main() {
	judgeAPI := api.StartAPI()
	judgeAPI.Run()
}

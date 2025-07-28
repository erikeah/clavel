package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/erikeah/clavel/internal/utils"
)

func main() {
	ctx := context.Background()
	results, err := utils.ParallelNixEval(ctx, os.Args[1])
	if err != nil {
		fmt.Println(err)
	} else {
		for result := range results {
			if data, err := json.Marshal(result); err == nil {
				fmt.Println(string(data))
			}
		}
	}
}

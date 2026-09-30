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
	results := utils.ParallelNixEval(ctx, os.Args[1:])
	for result := range results {
		if data, err := json.Marshal(result); err == nil {
			fmt.Println(string(data))
		}
	}
}

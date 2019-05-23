package logs

import (
	"fmt"
	"os"
	"sync"
	"text/tabwriter"
)

var scoreBoard []int
var scoreBoardLock sync.Mutex
var ScoreBoardLen int

func InitScoreBoard(maxRetryCount int) {
	scoreBoard = make([]int, maxRetryCount+2)
	ScoreBoardLen = len(scoreBoard)
}

func PrintScoreBoard() {
	if scoreBoard != nil {
		sum := 0
		fmt.Println("\n\nSubmission Results")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 0, ' ', tabwriter.Debug)
		for i, score := range scoreBoard {
			if i == 1 {
				fmt.Fprintf(w, "Finished with %d retry \t %d\n", i, score)
			} else if i < len(scoreBoard)-1 {
				fmt.Fprintf(w, "Finished with %d retries \t %d\n", i, score)
			} else {
				fmt.Fprintf(w, "Failed \t %d\n", scoreBoard[len(scoreBoard)-1])
			}
			sum += score
		}
		fmt.Fprintf(w, "TOTAL \t %d\n", sum)
		w.Flush()
	}
}

func IncrementScore(index int) {
	scoreBoardLock.Lock()
	defer scoreBoardLock.Unlock()
	if scoreBoard != nil && index >= 0 && index < len(scoreBoard) {
		scoreBoard[index]++
	}
}

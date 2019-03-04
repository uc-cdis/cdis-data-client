package logs

import (
	"fmt"
	"os"
	"sync"
	"text/tabwriter"
)

var ScoreBoard []int
var scoreBoardLock sync.Mutex

func InitScoreBoard(maxRetryCount int) {
	ScoreBoard = make([]int, maxRetryCount+2)
}

func PrintScoreBoard() {
	if ScoreBoard != nil {
		sum := 0
		fmt.Println("\n\nSubmission Results")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 0, ' ', tabwriter.Debug)
		for i, score := range ScoreBoard {
			if i < len(ScoreBoard)-1 {
				fmt.Fprintf(w, "Finished with %d retry/retries\t%d\n", i, score)
			} else {
				fmt.Fprintf(w, "Failed\t%d\n", ScoreBoard[len(ScoreBoard)-1])
			}
			sum += score
		}
		fmt.Fprintf(w, "TOTAL\t%d\n", sum)
		w.Flush()
	}
}

func IncrementScore(index int) {
	scoreBoardLock.Lock()
	defer scoreBoardLock.Unlock()
	if ScoreBoard != nil && index >= 0 && index < len(ScoreBoard) {
		ScoreBoard[index]++
	}
}

// Package agentproofreader puts a recognition right through the program a person already has
// installed: one question to the program for each batch, and the lines it would put right back.
package agentproofreader

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Proofreader asks the program about every batch of a recognition.
type Proofreader struct {
	// Answerer puts a question to the program.
	Answerer port.Answerer
	// Program is the program that answers, by the name it is listed under. Empty leaves the choice to
	// the providers.
	Program string
	// Command starts it. Empty starts the program where the provider finds it.
	Command []string
	// Model is which model corrects a recognition, by the name the program knows it as. Empty leaves
	// the choice to the program.
	Model string
	// Instruction is what the program is told it is doing. A scan and speech are corrected for
	// different mistakes, and the caller says which.
	Instruction string
	// InFlight is how many batches are asked about at once. It is how much of a person's own model is
	// taken while they are using it. Zero or less takes one.
	InFlight int

	// semaphore holds the turns at the program, taken before a question is put and given back after
	// it is answered, and is made the first time one is asked for.
	once      sync.Once
	semaphore chan struct{}
}

// acquire waits for a turn at the program, and hands back what gives it up.
func (p *Proofreader) acquire(ctx context.Context) (func(), error) {
	p.once.Do(func() {
		p.semaphore = make(chan struct{}, max(p.InFlight, 1))
	})
	select {
	case p.semaphore <- struct{}{}:
		return func() { <-p.semaphore }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// GetName is what put a correction right, recorded beside every line it made: the program, else
// "agent" where the providers choose it, and the model where one is named.
func (p *Proofreader) GetName() string {
	name := p.Program
	if name == "" {
		name = "agent"
	}
	if p.Model == "" {
		return name
	}
	return name + " " + p.Model
}

// Proofread asks about every batch and answers with what came back about each, by the batch it is
// about. A batch nothing came back about is left out.
//
// One batch that fails ends the run: the batches already answered are dropped and the caller asks
// again.
func (p *Proofreader) Proofread(ctx context.Context, batches []domain.ProofreadBatch) (map[int]string, error) {
	if len(batches) == 0 {
		return nil, nil
	}
	if p.Instruction == "" {
		return nil, errors.New("no instruction for the proofreading program")
	}

	ctx, stop := context.WithCancel(ctx)
	defer stop()

	var (
		mu     sync.Mutex
		out    = make(map[int]string, len(batches))
		failed error
		wg     sync.WaitGroup
	)
	for _, batch := range batches {
		turn, err := p.acquire(ctx)
		if err != nil {
			break
		}
		if ctx.Err() != nil {
			turn()
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer turn()
			reply, err := p.ask(ctx, batch)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err != nil:
				if failed == nil {
					failed = err
					stop()
				}
			case reply != "":
				out[batch.Number] = reply
			}
		}()
	}
	wg.Wait()

	if failed != nil {
		return nil, failed
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// ask puts one batch to the program and answers with what it wrote.
func (p *Proofreader) ask(ctx context.Context, batch domain.ProofreadBatch) (string, error) {
	reply, err := p.Answerer.Answer(ctx, port.Question{
		Program:      p.Program,
		Prompt:       batch.Prompt,
		Instructions: p.Instruction,
		Model:        p.Model,
		Command:      p.Command,
	})
	if err != nil {
		return "", fmt.Errorf("batch %d: %w", batch.Number, err)
	}
	return reply, nil
}

package cli_test

func (constantEmbedder) Concurrency() int { return 1 }

func (failing) Concurrency() int { return 1 }

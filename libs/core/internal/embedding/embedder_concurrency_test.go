package embedding_test

func (failing) Concurrency() int { return 1 }

func (landed) Concurrency() int { return 1 }

func (*shut) Concurrency() int { return 1 }

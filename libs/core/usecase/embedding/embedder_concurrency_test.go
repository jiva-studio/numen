package embedding_test

func (blocking) Concurrency() int { return 1 }

func (closing) Concurrency() int { return 1 }

func (counting) Concurrency() int { return 1 }

func (failing) Concurrency() int { return 1 }

func (*saying) Concurrency() int { return 1 }

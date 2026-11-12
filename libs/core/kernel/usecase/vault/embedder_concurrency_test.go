package vault_test

func (failing) Concurrency() int { return 1 }

func (pointing) Concurrency() int { return 1 }

func (unwilling) Concurrency() int { return 1 }

package search_test

func (*benchEmbedder) Concurrency() int { return 1 }

func (*countingEmbedder) Concurrency() int { return 1 }

func (oneWay) Concurrency() int { return 1 }

func (outOfReach) Concurrency() int { return 1 }

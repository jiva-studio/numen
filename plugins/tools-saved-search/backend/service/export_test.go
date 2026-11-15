package service

func (a *App) Wait() {
	a.parts.runner.Wait()
}

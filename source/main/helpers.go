package main

func (app *Application) registerService(uid string, store Store) {
	app.Mtx.Lock()
	defer app.Mtx.Unlock()
	app.Db[uid] = &store
}

func (app *Application) removeService(uid string) {
	app.Mtx.Lock()
	defer app.Mtx.Unlock()
	delete(app.Db, uid)
}

func (app *Application) getServices() []Store {
	var svs []Store
	app.Mtx.RLock()
	defer app.Mtx.RUnlock()
	for _, store := range app.Db {
		svs = append(svs, *store)
	}
	return svs
}

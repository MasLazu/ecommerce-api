package main

import "ecommerce-api/app"

func main() {
	app := app.NewApp()
	defer app.Database.CloseConn()
	app.Start()
}

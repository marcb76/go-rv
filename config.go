package main

// Global Configuration Constants for go-rv!
const (
	AppName              = "go-rv!"           // Official identification of the service
	Version              = "0.0.1"            // Version of the Proof of Concept
	InternalServerScheme = "http"             // Scheme used by the internal server (http or https)
	InternalServerHost   = "0.0.0.0"          // Host on which the internal server will run. 0.0.0.0 or "" binds to all available interfaces.
	InternalServerPort   = "8080"             // Port on which the internal server will run.
	ServerScheme         = "https"            // Scheme used by the server (http or https)
	ServerHost           = "go-rv.mbonet.xyz" // Server hostname or IP address where the server will listen
	ServerPort           = "443"              // Port on which the microservice will run.
)

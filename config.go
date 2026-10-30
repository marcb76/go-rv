package main

// Global Configuration Constants for go-rv!
const (
	AppName              = "go-rv!"                    // Official identification of the service
	Version              = "0.0.4"                     // Version of the Proof of Concept
	InternalServerScheme = "http"                      // Scheme used by the internal server (http or https)
	InternalServerHost   = "0.0.0.0"                   // Host on which the internal server will run. 0.0.0.0 or "" binds to all available interfaces.
	InternalServerPort   = "8080"                      // Port on which the internal server will run.
	ServerScheme         = "https"                     // Scheme used by the server (http or https)
	ServerHost           = "go-rv.mbonet.xyz"          // Server hostname or IP address where the server will listen
	ServerPort           = "443"                       // Port on which the microservice will run.
	CreateURLRateLimit   = 10                          // Maximum number of URL creation requests allowed per minute.
	GeminiAPIKeyEnv      = "GEMINI_API_KEY"            // Environment variable name for the Gemini API key
	MongoDBServer        = "rv-go.al2cc0d.mongodb.net" // MongoDB server address
	MongoDBName          = "go-rv"                     // MongoDB database name
	MongoDBCollection    = "short_urls"                // MongoDB collection name
	MongoDBUsername      = "go-rv"                     // MongoDB username
	MongoDBPasswordEnv   = "MONGODB_PASSWORD"          // Environment variable name for the MongoDB password
)

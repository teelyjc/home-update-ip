package constants

import "os"

var IsDevelopment = os.Getenv("ENVIRONMENT") != "production"

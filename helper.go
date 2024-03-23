package asyncjob

import "log"

func recovery() {
	if r := recover(); r != nil {
		log.Printf("error from recovery go runtime: %v", r)
	}
}

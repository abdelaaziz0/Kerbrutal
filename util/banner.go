package util

import "fmt"

func PrintBanner() {
	banner := `
    __             __               __        __
   / /_____  _____/ /_  _______  __/ /_____ _/ /
  / //_/ _ \/ ___/ __ \/ ___/ / / / __/ __ '/ / 
 / ,< /  __/ /  / /_/ / /  / /_/ / /_/ /_/ / /  
/_/|_|\___/_/  /_.___/_/   \__,_/\__/\__,_/_/   
`
	fmt.Printf("%v\nVersion: %v - %v\n\n", banner, Version, Author)
}

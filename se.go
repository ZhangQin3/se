package se

import (
	"se/selenium"
)

/* Element selectors */
const (
	ById = iota
	ByXPath
	ByLinkText
	ByPartialLinkText
	ByName
	ByTagName
	ByClassName
	ByCssSelector
	/* No responding string in the elemSelector */
	ByValue
	ByHerf
	ByTitle
	ByIndex
)

var elemSelector = []string{
	ById:              "id",
	ByXPath:           "xpath",
	ByLinkText:        "link text",
	ByPartialLinkText: "partial link text",
	ByName:            "name",
	ByTagName:         "tag name",
	ByClassName:       "class name",
	ByCssSelector:     "css selector",
}

/* Mouse buttons */
const (
	LeftButton = iota
	MiddleButton
	RightButton
)

/* Keys */
const (
	BackspaceKey = string(0x8)
	TabKey       = string(0x9)
	ClearKey     = string('\ue005')
	ReturnKey    = string(0x0A)
	EnterKey     = string(0x0A)
)

/* Unexported global variables. */
var (
	wd selenium.WebDriver
)

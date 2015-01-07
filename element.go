package se

import (
	"se/selenium"
	"strconv"
)

/* A preliminary element struct contains locators. */
type Element struct {
	page        *Page               // The page to which the Element belongs to.
	webElement  selenium.WebElement // The Element got form the page e.
	selector    string              // The element selector used to retrieve the element.
	selStrategy int                 // The element selector strategy, see: http://code.google.com/p/selenium/wiki/JsonWireProtocol#/session/:sessionId/element
}

func (e *Element) descendant(selector string) *Element {
	e.selector += " " + selector
	return e
}

func (e *Element) Tr(by int, selector interface{}) *Element {
	s := genCssSelector("tr", by, selector)
	return e.descendant(s)
}

func (e *Element) Td(by int, selector interface{}) *Element {
	s := genCssSelector("td", by, selector)
	return e.descendant(s)
}

func (e *Element) TextBox(by int, selector interface{}) *Element {
	s := genCssSelector("input", by, selector)
	return e.descendant(s)
}

func (e *Element) Link(by int, selector interface{}) *Element {
	s := genCssSelector("a", by, selector)
	return e.descendant(s)
}

func genCssSelector(tag string, by int, selector interface{}) (s string) {
	switch by {
	case ByCssSelector:
		s = selector.(string)
	case ById:
		s = tag + "#" + selector.(string)
	case ByClassName:
		s = tag + "." + selector.(string)
	case ByIndex:
		s = tag + ":nth-child(" + strconv.Itoa(selector.(int)) + ")"
	}
	return
}

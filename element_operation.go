package se

import (
	"se/selenium"
	// "gtf/drivers/log"
)

/* Click on element */
func (e *Element) Click() error {
	validateElement(e)
	return e.webElement.Click()
}

/* Send keys (type) into element */
func (e *Element) SendKeys(keys string) error {
	validateElement(e)
	return e.webElement.SendKeys(keys)
}

/* Set text into element */
func (e *Element) SetText(text string) error {
	validateElement(e)
	err := e.Clear()
	if err != nil {
		panic(err)
	}

	return e.webElement.SendKeys(text)
}

/* Submit */
func (e *Element) Submit() error {
	validateElement(e)
	return e.webElement.Submit()
}

/* Clear */
func (e *Element) Clear() error {
	validateElement(e)
	err := e.webElement.Clear()
	return err
}

/* Move mouse to relative coordinates */
func (e *Element) MoveTo(xOffset, yOffset int) error {
	validateElement(e)
	return e.webElement.MoveTo(xOffset, yOffset)
}

// Finding
/* Find children, return one element. */
func (e *Element) FindElement(by, value string) (selenium.WebElement, error) {
	validateElement(e)
	return e.webElement.FindElement(by, value)
}

/* Find children, return list of elements. */
func (e *Element) FindElements(by, value string) ([]selenium.WebElement, error) {
	validateElement(e)
	return e.webElement.FindElements(by, value)
}

// Porperties

/* Element name */
func (e *Element) TagName() (string, error) {
	validateElement(e)
	return e.webElement.TagName()
}

/* Text of element */
func (e *Element) Text() (string, error) {
	validateElement(e)
	return e.webElement.Text()
}

/* Check if element is selected. */
func (e *Element) IsSelected() (bool, error) {
	validateElement(e)
	return e.webElement.IsSelected()
}

/* Check if element is enabled. */
func (e *Element) IsEnabled() (bool, error) {
	validateElement(e)
	return e.webElement.IsEnabled()
}

/* Check if element is displayed. */
func (e *Element) IsDisplayed() (bool, error) {
	validateElement(e)
	return e.webElement.IsDisplayed()
}

/* Get element attribute. */
func (e *Element) GetAttribute(name string) (string, error) {
	validateElement(e)
	return e.webElement.GetAttribute(name)
}

/* Element location. */
func (e *Element) Location() (*selenium.Point, error) {
	validateElement(e)
	return e.webElement.Location()
}

/* Element location once it has been scrolled into view. */
func (e *Element) LocationInView() (*selenium.Point, error) {
	validateElement(e)
	return e.webElement.LocationInView()
}

/* Element size */
func (e *Element) Size() (*selenium.Size, error) {
	validateElement(e)
	return e.webElement.Size()
}

/* Get element CSS property value. */
func (e *Element) CssProperty(name string) (string, error) {
	validateElement(e)
	return e.webElement.CSSProperty(name)
}

/* Check if element exists. */
func (e *Element) DoesExist() (bool, error) {
	if e.webElement == nil {
		_, err := e.page.webDriver.FindElement(elemSelector[e.selStrategy], e.selector)
		if q, ok := err.(*selenium.QueryError); ok && q.Status == 7 {
			return false, err
		}

		if err != nil {
			panic(err)
		}
	}
	return true, nil
}

func validateElement(e *Element) {
	if e.webElement == nil {
		elem, err := e.page.webDriver.FindElement(elemSelector[e.selStrategy], e.selector)
		if err != nil {
			panic(err)
		}
		e.webElement = elem
	}
}

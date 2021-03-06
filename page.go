// https://github.com/SeleniumHQ/selenium/wiki/JsonWireProtocol
// https://github.com/SeleniumHQ/selenium/wiki/DesiredCapabilities
package se

import (
	"gtf/drivers/log"
	"se/selenium"
	"strings"
)

/* Page interface implementation */
type Page struct {
	webDriver selenium.WebDriver
	url       string
}

/* Here, the returned type is struct{ Page }, seems tricky, it is used to compatible to the
customed package, and maky it easy to write costomed package.
Due to some package may has "type GwLoginPage struct{ Page }" definition to promote methods of webgui
*/
func OpenPage(url string, wd selenium.WebDriver, params ...func(caps map[string]interface{})) (s struct{ Page }, err error) {
	if wd == nil {
		caps := selenium.Capabilities{"browserName": "chrome", "takesScreenshot": true} //{android|chrome|firefox|htmlunit|internet explorer|iPhone|iPad|opera|safari}.
		for _, f := range params {
			f(caps)
		}
		wd, err = selenium.NewRemote(caps, "")
		if err != nil {
			panic(err)
		}

		wd.MaximizeWindow("current")
	}

	p := Page{webDriver: wd, url: url}
	// p.webDriver.SetImplicitWaitTimeout(100)
	// p.webDriver.SetAsyncScriptTimeout(1000)
	// p.webDriver.SetTimeout("script", 100)
	// p.webDriver.SetTimeout("implicit", 100)
	p.webDriver.SetTimeout("page load", 15000)
	err = p.Open()
	return struct{ Page }{p}, err
}

func Browsername(name string) func(caps map[string]interface{}) {
	return func(caps map[string]interface{}) {
		caps["browserName"] = name
	}
}

func Screenshot(b bool) func(caps map[string]interface{}) {
	return func(caps map[string]interface{}) {
		caps["browserName"] = b
	}
}

// func genPage(in []interface{}) struct{ Page } {
// 	var wd selenium.WebDriver
// 	if len(in) > 0 {
// 		wd = in[0].(selenium.WebDriver)
// 	}
// 	if wd == nil {
// 		caps := selenium.Capabilities{"browserName": "firefox"}
// 		wd, _ = selenium.NewRemote(caps, "")
// 	}
// 	page := struct{ Page }{Page{wd: wd}}

// 	return page
// }

/* Open page against url. */
func (p *Page) Open() error {
	return p.webDriver.Get(p.url)
}

/* Close current window. */
func (p *Page) Close() error {
	return p.webDriver.Close()
}

/* Quit (end) current session */
func (p *Page) Quit() error {
	return p.webDriver.Quit()
}

func (p *Page) FindElement(by int, selector string) (*Element, error) {
	elem, err := p.webDriver.FindElement(elemSelector[by], selector)
	if err != nil {
		log.Warning(">>>>>>>>>>>>>>>>>>>>>>>>>>>")
		log.Warning(err)
		log.Warning("<<<<<<<<<<<<<<<<<<<<<<<<<<<")
		return nil, err
	}

	return &Element{page: p, webElement: elem}, nil
}

func (p *Page) FindElementAndClick(by int, selector string) *Element {
	elem, err := p.webDriver.FindElement(elemSelector[by], selector)
	if err != nil {
		log.Warning(">>>>>>>>>>>>>>>>>>>>>>>>>>>")
		log.Warning(err)
		log.Warning("<<<<<<<<<<<<<<<<<<<<<<<<<<<")
	}
	elem.Click()
	return &Element{page: p, webElement: elem}
}

func (p *Page) Element(tag string, by int, selector, auxSelector string) *Element {
	var s string
	e := new(Element)
	e.selStrategy = ByCssSelector
	switch by {
	case ByCssSelector:
		s = selector
	case ById:
		s = tag + "#" + selector
	case ByClassName:
		s = tag + "." + strings.Replace(selector, " ", ".", -1)
	case ByName:
		s = tag + "[name='" + selector + "']"
	case ByValue:
		s = tag + "[value='" + selector + "']"
	case ByHerf:
		s = tag + "[href='" + selector + "']"
	case ByPartialLinkText, ByLinkText:
		e.selStrategy = ByPartialLinkText
		s = selector
	}

	if auxSelector != "" {
		s += auxSelector
	}
	e.page, e.selector = p, s
	return e
}

func (p *Page) Div(by int, selector string) *Element {
	return p.Element("div", by, selector, "")
}

func (p *Page) Link(by int, selector string) *Element {
	return p.Element("a", by, selector, "")
}

func (p *Page) Form(by int, selector string) *Element {
	return p.Element("form", by, selector, "")
}

func (p *Page) Button(by int, selector string) *Element {
	return p.Element("input", by, selector, "[type=button]")
}

func (p *Page) SubmitBtn(by int, selector string) *Element {
	return p.Element("input", by, selector, "[type=submit]")
}

func (p *Page) TextBox(by int, selector string) *Element {
	return p.Element("input", by, selector, "[type=text]")
}

func (p *Page) PasswordBox(by int, selector string) *Element {
	return p.Element("input", by, selector, "[type=password]")
}

func (p *Page) CheckBox(by int, selector string) *Element {
	return p.Element("input", by, selector, "[type=checkbox]")
}

func (p *Page) Table(by int, selector string) *Element {
	return p.Element("table", by, selector, "")
}

func (p *Page) ScreenShot() (string, error) {
	return p.webDriver.Screenshot()
}

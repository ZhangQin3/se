/* Remote Selenium client implementation.
See http://code.google.com/p/selenium/wiki/JsonWireProtocol for wire protocol.
*/

package selenium

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"gtf/drivers/log"
	"io/ioutil"
	"net/http"
	// "net/http/httputil"
	"net/url"
	"regexp"
	"strings"
)

/* Errors returned by Selenium server. */
var errors_ = map[int]string{
	0:  "success",
	7:  "no such element",
	8:  "no such frame",
	9:  "unknown command",
	10: "stale element reference",
	11: "element not visible",
	12: "invalid element state",
	13: "unknown error",
	15: "element is not selectable",
	17: "javascript error",
	19: "xpath lookup error",
	21: "timeout",
	23: "no such window",
	24: "invalid cookie domain",
	25: "unable to set cookie",
	26: "unexpected alert open",
	27: "no alert open",
	28: "script timeout",
	29: "invalid element coordinates",
	32: "invalid selector",
}

const (
	SUCCESS          = 0
	DEFAULT_EXECUTOR = "http://127.0.0.1:4444/wd/hub"
	JSON_MIME_TYPE   = "application/json"
	MAX_REDIRECTS    = 10
)

type remoteWD struct {
	id, executor string
	capabilities Capabilities
	// FIXME
	// profile             BrowserProfile
}

type serverReply struct {
	SessionId *string // sessionId can be null
	Status    int
	Message   string
	Value     json.RawMessage
}

type value struct {
	Screen  json.RawMessage
	Message string
	Text    string
}

type statusReply struct {
	Value  Status
	Status int
}
type stringReply struct {
	Value  *string
	Status int
}
type stringsReply struct {
	Value  []string
	Status int
}
type boolReply struct {
	Value  bool
	Status int
}
type element struct {
	ELEMENT string
	// Screen  *string
}
type elementReply struct {
	Value  element
	Status int
}
type elementsReply struct {
	Value  []element
	Status int
}
type cookiesReply struct {
	Value  []Cookie
	Status int
}
type locationReply struct {
	Value  Point
	Status int
}
type sizeReply struct {
	Value  Size
	Status int
}
type anyReply struct {
	Value  interface{}
	Status int
}
type capabilitiesReply struct {
	Value  Capabilities
	Status int
}

var httpClient *http.Client

func GetHTTPClient() *http.Client {
	return httpClient
}

func isMimeType(response *http.Response, mtype string) bool {
	if ctype, ok := response.Header["Content-Type"]; ok {
		return strings.HasPrefix(ctype[0], mtype)
	}

	return false
}

func cleanNils(buf []byte) {
	for i, b := range buf {
		if b == 0 {
			buf[i] = ' '
		}
	}
}

func isRedirect(response *http.Response) bool {
	switch response.StatusCode {
	case 301, 302, 303, 307:
		return true
	}
	return false
}

func normalizeURL(n string, base string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf(
			"Failed to parse base URL %s with error %s", base, err)
	}
	nURL, err := baseURL.Parse(n)
	if err != nil {
		return "", fmt.Errorf("Failed to parse new URL %s with error %s", n, err)
	}
	return nURL.String(), nil
}

func (wd *remoteWD) requestURL(template string, args ...interface{}) string {
	path := fmt.Sprintf(template, args...)
	return wd.executor + path
}

var reg = regexp.MustCompile(`: {\\"method\\":.+?"screen":.+?}`)

func (wd *remoteWD) execute(method, url string, data []byte) ([]byte, error) {
	// Trace := false
	debugLog("-> %s, %s", method, url)
	log.ToggleText("Application json", string(data), "off")
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", JSON_MIME_TYPE)
	if method == "POST" {
		req.Header.Add("Content-Type", JSON_MIME_TYPE)
	}

	// if Trace {
	// 	if dump, err := httputil.DumpRequest(req, true); err == nil && log != nil {
	// 		log.Printf("-> TRACE\n%s", dump)
	// 	}
	// }

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// if Trace {
	// 	if dump, err := httputil.DumpResponse(res, true); err == nil && log != nil {
	// 		log.Printf("<- TRACE\n%s", dump)
	// 	}
	// }

	buf, err := ioutil.ReadAll(res.Body)
	// debugLog("<- %s, %s", res.Status, res.Header["Content-Type"])
	// log.ToggleText("Application json", string(reg.ReplaceAll(buf, nil)), "off")
	if err != nil {
		buf = []byte(res.Status)
		return nil, errors.New(string(buf))
	}

	reply := new(serverReply)
	err = json.Unmarshal(buf, reply)
	if err != nil {
		return nil, fmt.Errorf(`{"message":"%s"}`, err)
	}

	v := new(value)
	if err = json.Unmarshal(reply.Value, v); err == nil && v.Screen != nil {
		var s string
		if err = json.Unmarshal(v.Screen, &s); err != nil {
			log.Infof("Unmarshal reply.Value.Screen failed: %s", err)
		}
		logScreenShot(&s)
	}

	cleanNils(buf)
	state, ok := errors_[reply.Status]
	if !ok {
		state = fmt.Sprintf("unknown error - %d", reply.Status)
	}
	if res.StatusCode >= 400 {
		warningLog("<--- %s, %s\n", res.Status, state)
		return nil, &QueryError{Status: reply.Status, Message: state}
	} else {
		debugLog("<- %s, %s\n", res.Status, state)
	}

	if isMimeType(res, JSON_MIME_TYPE) {
		if reply.Status != SUCCESS {
			message, ok := errors_[reply.Status]
			if !ok {
				message = fmt.Sprintf("unknown error - %d", reply.Status)
			}
			return nil, fmt.Errorf(`{"status":%d, "message":"%s"}`, reply.Status, message)
		}
	}
	return buf, nil
}

/* Create new remote client, this will also start a new session.
   capabilities - the desired capabilities, see http://goo.gl/SNlAk
   executor - the URL to the Selenim server, *must* be prefixed with protocol (http,https...).

   Empty string means DEFAULT_EXECUTOR
*/
func NewRemote(capabilities Capabilities, executor string) (WebDriver, error) {
	if len(executor) == 0 {
		executor = DEFAULT_EXECUTOR
	}

	wd := &remoteWD{executor: executor, capabilities: capabilities}
	// FIXME: Handle profile

	_, err := wd.NewSession()
	if err != nil {
		return nil, err
	}

	return wd, nil
}

func (wd *remoteWD) stringCommand(urlTemplate string) (string, error) {
	url := wd.requestURL(urlTemplate, wd.id)
	response, err := wd.execute("GET", url, nil)
	if err != nil {
		return "", err
	}

	reply := new(stringReply)
	err = json.Unmarshal(response, reply)
	if err != nil {
		return "", fmt.Errorf(`{"message":"%s"}`, err.Error())
	}
	if reply.Status != SUCCESS {
		message, ok := errors_[reply.Status]
		if !ok {
			message = fmt.Sprintf("unknown error - %d", reply.Status)
		}
		return "", fmt.Errorf(`{"status":%d, "message":"%s"}`, reply.Status, message)
	}

	if reply.Value == nil {
		return "", fmt.Errorf(`{"message":nil return value"}`)
	}

	return *reply.Value, nil
}

func (wd *remoteWD) voidCommand(urlTemplate string, params interface{}) (err error) {
	var data []byte
	if params != nil {
		data, err = json.Marshal(params)
	}
	if err == nil {
		_, err = wd.execute("POST", wd.requestURL(urlTemplate, wd.id), data)
	}
	return

}

func (wd *remoteWD) stringsCommand(urlTemplate string) ([]string, error) {
	url := wd.requestURL(urlTemplate, wd.id)
	response, err := wd.execute("GET", url, nil)
	if err != nil {
		return nil, err
	}
	reply := new(stringsReply)
	err = json.Unmarshal(response, reply)
	if err != nil {
		return nil, fmt.Errorf(`{"message":"%s"}`, err.Error())
	}

	if reply.Status != SUCCESS {
		message, ok := errors_[reply.Status]
		if !ok {
			message = fmt.Sprintf("unknown error - %d", reply.Status)
		}
		return nil, fmt.Errorf(`{"status":%d, "message":"%s"}`, reply.Status, message)
	}

	return reply.Value, nil
}

func (wd *remoteWD) boolCommand(urlTemplate string) (bool, error) {
	url := wd.requestURL(urlTemplate, wd.id)
	response, err := wd.execute("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf(`{"message":"%s"}`, err.Error())
	}

	reply := new(boolReply)
	err = json.Unmarshal(response, reply)
	if err != nil {
		return false, fmt.Errorf(`{"message":"%s"}`, err.Error())
	}

	message, ok := errors_[reply.Status]
	if !ok {
		message = fmt.Sprintf("unknown error - %d", reply.Status)
	}
	return false, fmt.Errorf(`{"status":%d, "message":"%s"}`, reply.Status, message)

	return reply.Value, nil
}

// WebDriver interface implementation

func (wd *remoteWD) Status() (*Status, error) {
	url := wd.requestURL("/status")
	reply, err := wd.execute("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf(`{"message":"%s"}`, err.Error())
	}

	status := new(statusReply)
	err = json.Unmarshal(reply, status)
	if err != nil {
		return nil, fmt.Errorf(`{"message":"%s"}`, err.Error())
	}

	if status.Status != SUCCESS {
		message, ok := errors_[status.Status]
		if !ok {
			message = fmt.Sprintf("unknown error - %d", status.Status)
		}
		return nil, fmt.Errorf(`{"status":%d, "message":"%s"}`, status.Status, message)
	}

	return &status.Value, nil
}

func (wd *remoteWD) NewSession() (string, error) {
	message := map[string]interface{}{
		"sessionId":           nil,
		"desiredCapabilities": wd.capabilities,
	}
	data, err := json.Marshal(message)
	if err != nil {
		return "", nil
	}

	url := wd.requestURL("/session")
	response, err := wd.execute("POST", url, data)
	if err != nil {
		return "", fmt.Errorf(`{"message":"%s"}`, err.Error())
	}

	reply := new(serverReply)
	json.Unmarshal(response, reply)

	wd.id = *reply.SessionId

	return wd.id, nil
}

func (wd *remoteWD) SessionId() string {
	return wd.id
}

func (wd *remoteWD) Capabilities() (Capabilities, error) {
	url := wd.requestURL("/session/%s", wd.id)
	response, err := wd.execute("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf(`{"message":"%s"}`, err.Error())
	}

	c := new(capabilitiesReply)
	err = json.Unmarshal(response, c)
	if err != nil {
		return nil, fmt.Errorf(`{"message":"%s"}`, err.Error())
	}

	if c.Status != SUCCESS {
		message, ok := errors_[c.Status]
		if !ok {
			message = fmt.Sprintf("unknown error - %d", c.Status)
		}
		return nil, fmt.Errorf(`{"status":%d, "message":"%s"}`, c.Status, message)
	}

	return c.Value, nil
}

// timeoutType - {string} The type of operation to set the timeout for. Valid values are: "script" for script timeouts, "implicit" for modifying the implicit wait timeout and "page load" for setting a page load timeout.
// ms          - {number} The amount of time, in milliseconds, that time-limited commands are permitted to run.
func (wd *remoteWD) SetTimeout(timeoutType string, ms uint) error {
	params := map[string]interface{}{"type": timeoutType, "ms": ms}
	return wd.voidCommand("/session/%s/timeouts", params)
}

func (wd *remoteWD) SetAsyncScriptTimeout(ms uint) error {
	params := map[string]uint{"ms": ms}
	return wd.voidCommand("/session/%s/timeouts/async_script", params)
}

func (wd *remoteWD) SetImplicitWaitTimeout(ms uint) error {
	params := map[string]uint{"ms": ms}
	return wd.voidCommand("/session/%s/timeouts/implicit_wait", params)
}

func (wd *remoteWD) AvailableEngines() ([]string, error) {
	return wd.stringsCommand("/session/%s/ime/available_engines")
}

func (wd *remoteWD) ActiveEngine() (string, error) {
	return wd.stringCommand("/session/%s/ime/active_engine")
}

func (wd *remoteWD) IsEngineActivated() (bool, error) {
	return wd.boolCommand("/session/%s/ime/activated")
}

func (wd *remoteWD) DeactivateEngine() error {
	return wd.voidCommand("session/%s/ime/deactivate", nil)
}

func (wd *remoteWD) ActivateEngine(engine string) error {
	params := map[string]string{"engine": engine}
	return wd.voidCommand("/session/%s/ime/activate", params)
}

func (wd *remoteWD) Quit() error {
	url := wd.requestURL("/session/%s", wd.id)
	_, err := wd.execute("DELETE", url, nil)
	if err == nil {
		wd.id = ""
	}

	return err
}

func (wd *remoteWD) CurrentWindowHandle() (string, error) {
	return wd.stringCommand("/session/%s/window_handle")
}

func (wd *remoteWD) WindowHandles() ([]string, error) {
	return wd.stringsCommand("/session/%s/window_handles")
}

func (wd *remoteWD) CurrentURL() (string, error) {
	url := wd.requestURL("/session/%s/url", wd.id)
	response, err := wd.execute("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf(`{"message":"%s"}`, err.Error())
	}
	reply := new(stringReply)
	json.Unmarshal(response, reply)

	return *reply.Value, nil

}

func (wd *remoteWD) Get(url string) error {
	requestURL := wd.requestURL("/session/%s/url", wd.id)
	params := map[string]string{
		"url": url,
	}
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf(`{"message":"%s"}`, err.Error())
	}

	_, err = wd.execute("POST", requestURL, data)

	if err != nil {
		return fmt.Errorf(`{"message":"%s"}`, err.Error())
	}
	return nil
}

func (wd *remoteWD) Forward() error {
	return wd.voidCommand("/session/%s/forward", nil)
}

func (wd *remoteWD) Back() error {
	return wd.voidCommand("/session/%s/back", nil)
}

func (wd *remoteWD) Refresh() error {
	return wd.voidCommand("/session/%s/refresh", nil)
}

func (wd *remoteWD) Title() (string, error) {
	return wd.stringCommand("/session/%s/title")
}

func (wd *remoteWD) PageSource() (string, error) {
	return wd.stringCommand("/session/%s/source")
}

func (wd *remoteWD) find(by, value, suffix, url string) ([]byte, error) {
	params := map[string]string{
		"using": by,
		"value": value,
	}
	data, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	if len(url) == 0 {
		url = "/session/%s/element"
	}

	urlTemplate := url + suffix
	url = wd.requestURL(urlTemplate, wd.id)
	return wd.execute("POST", url, data)
}

func (wd *remoteWD) DecodeElement(data []byte) (WebElement, error) {
	reply := new(elementReply)
	err := json.Unmarshal(data, reply)
	if err != nil {
		return nil, fmt.Errorf(`{"message":"%s"}`, err.Error())
	}

	if reply.Status != SUCCESS {
		message, ok := errors_[reply.Status]
		if !ok {
			message = fmt.Sprintf("unknown error - %d", reply.Status)
		}
		return nil, fmt.Errorf(`{"status":%d, "message":"%s"}`, reply.Status, message)
	}

	elem := &remoteWE{wd, reply.Value.ELEMENT}
	return elem, nil
}

func (wd *remoteWD) FindElement(by, value string) (WebElement, error) {
	response, err := wd.find(by, value, "", "")
	if err != nil {
		return nil, err
	}

	return wd.DecodeElement(response)
}

func (wd *remoteWD) DecodeElements(data []byte) ([]WebElement, error) {
	reply := new(elementsReply)
	err := json.Unmarshal(data, reply)
	if err != nil {
		return nil, fmt.Errorf(`{"message":"%s"}`, err.Error())
	}

	elems := make([]WebElement, len(reply.Value))
	for i, elem := range reply.Value {
		elems[i] = &remoteWE{wd, elem.ELEMENT}
	}

	return elems, nil
}

func (wd *remoteWD) FindElements(by, value string) ([]WebElement, error) {
	response, err := wd.find(by, value, "s", "")
	if err != nil {
		return nil, err
	}

	return wd.DecodeElements(response)
}

func (wd *remoteWD) Close() error {
	url := wd.requestURL("/session/%s/window", wd.id)
	_, err := wd.execute("DELETE", url, nil)
	return err
}

func (wd *remoteWD) SwitchWindow(name string) error {
	params := map[string]string{"name": name}
	return wd.voidCommand("/session/%s/window", params)
}

func (wd *remoteWD) CloseWindow(name string) error {
	url := wd.requestURL("/session/%s/window", wd.id)
	_, err := wd.execute("DELETE", url, nil)
	return err
}

func (wd *remoteWD) MaximizeWindow(name string) error {
	var err error
	if len(name) == 0 {
		name, err = wd.CurrentWindowHandle()
		if err != nil {
			return err
		}
	}

	url := wd.requestURL("/session/%s/window/%s/maximize", wd.id, name)
	_, err = wd.execute("POST", url, nil)
	return err
}

func (wd *remoteWD) SwitchFrame(frame string) error {
	params := map[string]string{"id": frame}
	return wd.voidCommand("/session/%s/frame", params)
}

func (wd *remoteWD) ActiveElement() (WebElement, error) {
	url := wd.requestURL("/session/%s/element/active", wd.id)
	response, err := wd.execute("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return wd.DecodeElement(response)
}

func (wd *remoteWD) GetCookies() ([]Cookie, error) {
	url := wd.requestURL("/session/%s/cookie", wd.id)
	data, err := wd.execute("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf(`{"message":"%s"}`, err.Error())
	}

	reply := new(cookiesReply)
	err = json.Unmarshal(data, reply)
	if err != nil {
		return nil, fmt.Errorf(`{"message":"%s"}`, err.Error())
	}

	return reply.Value, nil
}

func (wd *remoteWD) AddCookie(cookie *Cookie) error {
	params := map[string]*Cookie{"cookie": cookie}
	return wd.voidCommand("/session/%s/cookie", params)
}

func (wd *remoteWD) DeleteAllCookies() error {
	url := wd.requestURL("/session/%s/cookie", wd.id)
	_, err := wd.execute("DELETE", url, nil)
	return err
}

func (wd *remoteWD) DeleteCookie(name string) error {
	url := wd.requestURL("/session/%s/cookie/%s", wd.id, name)
	_, err := wd.execute("DELETE", url, nil)
	return err
}

func (wd *remoteWD) Click(button int) error {
	params := map[string]int{"button": button}
	return wd.voidCommand("/session/%s/click", params)
}

func (wd *remoteWD) DoubleClick() error {
	return wd.voidCommand("/session/%s/doubleclick", nil)
}

func (wd *remoteWD) ButtonDown() error {
	return wd.voidCommand("/session/%s/buttondown", nil)
}

func (wd *remoteWD) ButtonUp() error {
	return wd.voidCommand("/session/%s/buttonup", nil)
}

func (wd *remoteWD) SendModifier(modifier string, isDown bool) error {
	params := map[string]interface{}{"value": modifier, "isdown": isDown}
	return wd.voidCommand("/session/%s/modifier", params)
}

func (wd *remoteWD) DismissAlert() error {
	return wd.voidCommand("/session/%s/dismiss_alert", nil)
}

func (wd *remoteWD) AcceptAlert() error {
	return wd.voidCommand("/session/%s/accept_alert", nil)
}

func (wd *remoteWD) AlertText() (string, error) {
	return wd.stringCommand("/session/%s/alert_text")
}

func (wd *remoteWD) SetAlertText(text string) error {
	params := map[string]string{"text": text}
	return wd.voidCommand("/session/%s/alert_text", params)
}

func (wd *remoteWD) execScriptRaw(script string, args []interface{}, suffix string) ([]byte, error) {
	params := map[string]interface{}{
		"script": script,
		"args":   args,
	}

	data, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	template := "/session/%s/execute" + suffix
	url := wd.requestURL(template, wd.id)
	return wd.execute("POST", url, data)
}

func (wd *remoteWD) execScript(script string, args []interface{}, suffix string) (interface{}, error) {
	response, err := wd.execScriptRaw(script, args, suffix)
	if err != nil {
		return nil, fmt.Errorf(`{"message":"%s"}`, err.Error())
	}

	reply := new(anyReply)
	err = json.Unmarshal(response, reply)
	if err != nil {
		return nil, fmt.Errorf(`{"message":"%s"}`, err.Error())
	}

	return reply.Value, nil
}

func (wd *remoteWD) ExecuteScript(script string, args []interface{}) (interface{}, error) {
	return wd.execScript(script, args, "")
}

func (wd *remoteWD) ExecuteScriptAsync(script string, args []interface{}) (interface{}, error) {
	return wd.execScript(script, args, "_async")
}

func (wd *remoteWD) ExecuteScriptRaw(script string, args []interface{}) ([]byte, error) {
	return wd.execScriptRaw(script, args, "")
}

func (wd *remoteWD) ExecuteScriptAsyncRaw(script string, args []interface{}) ([]byte, error) {
	return wd.execScriptRaw(script, args, "_async")
}

func (wd *remoteWD) Screenshot() (string, error) {
	data, err := wd.stringCommand("/session/%s/screenshot")
	if err != nil {
		return "", err
	}

	return logScreenShot(&data)
}

// WebElement interface implementation

type remoteWE struct {
	parent *remoteWD
	id     string
}

func (elem *remoteWE) Click() error {
	urlTemplate := fmt.Sprintf("/session/%%s/element/%s/click", elem.id)
	return elem.parent.voidCommand(urlTemplate, nil)
}

func (elem *remoteWE) SendKeys(keys string) error {
	chars := make([]string, len(keys))
	for i, c := range keys {
		chars[i] = string(c)
	}
	params := map[string][]string{"value": chars}
	urlTemplate := fmt.Sprintf("/session/%%s/element/%s/value", elem.id)
	return elem.parent.voidCommand(urlTemplate, params)
}

func (elem *remoteWE) TagName() (string, error) {
	urlTemplate := fmt.Sprintf("/session/%%s/element/%s/name", elem.id)
	return elem.parent.stringCommand(urlTemplate)
}

func (elem *remoteWE) Text() (string, error) {
	urlTemplate := fmt.Sprintf("/session/%%s/element/%s/text", elem.id)
	return elem.parent.stringCommand(urlTemplate)
}

func (elem *remoteWE) Submit() error {
	urlTemplate := fmt.Sprintf("/session/%%s/element/%s/submit", elem.id)
	return elem.parent.voidCommand(urlTemplate, nil)
}

func (elem *remoteWE) Clear() error {
	urlTemplate := fmt.Sprintf("/session/%%s/element/%s/clear", elem.id)
	return elem.parent.voidCommand(urlTemplate, nil)
}

func (elem *remoteWE) MoveTo(xOffset, yOffset int) error {
	params := map[string]interface{}{"element": elem.id, "xoffset": xOffset, "yoffset": yOffset}
	return elem.parent.voidCommand("/session/%s/moveto", params)
}

func (elem *remoteWE) FindElement(by, value string) (WebElement, error) {
	url := fmt.Sprintf("/session/%%s/element/%s/element", elem.id)
	response, err := elem.parent.find(by, value, "", url)
	if err != nil {
		return nil, err
	}

	return elem.parent.DecodeElement(response)
}

func (elem *remoteWE) FindElements(by, value string) ([]WebElement, error) {
	url := fmt.Sprintf("/session/%%s/element/%s/element", elem.id)
	response, err := elem.parent.find(by, value, "s", url)
	if err != nil {
		return nil, err
	}

	return elem.parent.DecodeElements(response)
}

func (elem *remoteWE) boolQuery(urlTemplate string) (bool, error) {
	url := fmt.Sprintf(urlTemplate, elem.id)
	return elem.parent.boolCommand(url)
}

// Porperties
func (elem *remoteWE) IsSelected() (bool, error) {
	return elem.boolQuery("/session/%%s/element/%s/selected")
}

func (elem *remoteWE) IsEnabled() (bool, error) {
	return elem.boolQuery("/session/%%s/element/%s/enabled")
}

func (elem *remoteWE) IsDisplayed() (bool, error) {
	return elem.boolQuery("/session/%%s/element/%s/displayed")
}

func (elem *remoteWE) GetAttribute(name string) (string, error) {
	template := "/session/%%s/element/%s/attribute/%s"
	urlTemplate := fmt.Sprintf(template, elem.id, name)

	return elem.parent.stringCommand(urlTemplate)
}

func (elem *remoteWE) location(suffix string) (*Point, error) {
	wd := elem.parent
	path := "/session/%s/element/%s/location" + suffix
	url := wd.requestURL(path, wd.id, elem.id)
	response, err := wd.execute("GET", url, nil)
	if err != nil {
		return nil, err
	}
	reply := new(locationReply)
	err = json.Unmarshal(response, reply)
	if err != nil {
		return nil, fmt.Errorf(`{"message":"%s"}`, err.Error())
	}

	return &reply.Value, nil
}

func (elem *remoteWE) Location() (*Point, error) {
	return elem.location("")
}

func (elem *remoteWE) LocationInView() (*Point, error) {
	return elem.location("_in_view")
}

func (elem *remoteWE) Size() (*Size, error) {
	wd := elem.parent
	url := wd.requestURL("/session/%s/element/%s/size", wd.id, elem.id)
	response, err := wd.execute("GET", url, nil)
	if err != nil {
		return nil, err
	}
	reply := new(sizeReply)
	err = json.Unmarshal(response, reply)
	if err != nil {
		return nil, fmt.Errorf(`{"message":"%s"}`, err.Error())
	}

	return &reply.Value, nil
}

func (elem *remoteWE) CSSProperty(name string) (string, error) {
	wd := elem.parent
	urlTemplate := fmt.Sprintf("/session/%s/element/%s/css/%s", wd.id, elem.id, name)
	return elem.parent.stringCommand(urlTemplate)
}

func init() {
	httpClient = &http.Client{
		// WebDriver requires that all requests have an 'Accept: application/json' header. We must add
		// it here because by default net/http will not include that header when following redirects.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > MAX_REDIRECTS {
				return fmt.Errorf("too many redirects (%d)", len(via))
			}
			req.Header.Add("Accept", JSON_MIME_TYPE)
			// if Trace {
			// 	if dump, err := httputil.DumpRequest(req, true); err == nil && Log != nil {
			// 		Log.Printf("-> TRACE (redirected request)\n%s", dump)
			// 	}
			// }
			return nil
		},
	}
}

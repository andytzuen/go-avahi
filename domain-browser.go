package avahi

import (
	"fmt"

	dbus "github.com/godbus/dbus/v5"
)

// A DomainBrowser is used to browse for mDNS domains
type DomainBrowser struct {
	object        dbus.BusObject
	AddChannel    chan Domain
	RemoveChannel chan Domain
	closeCh       chan struct{}
}

const (
	// DomainBrowserTypeBrowse - Browse for a list of available browsing domains
	DomainBrowserTypeBrowse = 0
	// DomainBrowserTypeBrowseDefault - Browse for the default browsing domain
	DomainBrowserTypeBrowseDefault = 1
	// DomainBrowserTypeRegister - Browse for a list of available registering domains
	DomainBrowserTypeRegister = 2
	// DomainBrowserTypeRegisterDefault - Browse for the default registering domain
	DomainBrowserTypeRegisterDefault = 3
	// DomainBrowserTypeBrowseLegacy - Legacy browse domain - see DNS-SD spec for more information
	DomainBrowserTypeBrowseLegacy = 4
)

// DomainBrowserNew returns a new domain browser
func DomainBrowserNew(conn *dbus.Conn, path dbus.ObjectPath) (*DomainBrowser, error) {
	c := new(DomainBrowser)

	c.object = conn.Object("org.freedesktop.Avahi", path)
	c.AddChannel = make(chan Domain)
	c.RemoveChannel = make(chan Domain)
	c.closeCh = make(chan struct{})

	return c, nil
}

func (c *DomainBrowser) interfaceForMember(method string) string {
	return fmt.Sprintf("%s.%s", "org.freedesktop.Avahi.DomainBrowser", method)
}

func (c *DomainBrowser) free() {
	close(c.closeCh)
	c.object.Call(c.interfaceForMember("Free"), 0)
}

func (c *DomainBrowser) getObjectPath() dbus.ObjectPath {
	return c.object.Path()
}

func (c *DomainBrowser) dispatchSignal(signal *dbus.Signal) error {
	if signal.Name == c.interfaceForMember("ItemNew") || signal.Name == c.interfaceForMember("ItemRemove") {
		var domain Domain
		err := dbus.Store(signal.Body, &domain.Interface, &domain.Protocol, &domain.Domain, &domain.Flags)
		if err != nil {
			return err
		}

		if signal.Name == c.interfaceForMember("ItemNew") {
			select {
			case c.AddChannel <- domain:
			case <-c.closeCh:
				close(c.AddChannel)
				close(c.RemoveChannel)
			}
		} else {
			select {
			case c.RemoveChannel <- domain:
			case <-c.closeCh:
				close(c.AddChannel)
				close(c.RemoveChannel)
			}
		}
	}

	return nil
}

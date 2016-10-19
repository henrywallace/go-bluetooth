package bluez

import (
	"github.com/godbus/dbus"
	"github.com/muka/device-manager/client"
	"log"
)

// NewObjectManager create a new Device1 client
func NewObjectManager() *ObjectManager {
	a := new(ObjectManager)
	a.client = client.NewClient(
		&client.Config{
			Name:  "org.bluez",
			Iface: "org.freedesktop.DBus.ObjectManager",
			Path:  "/",
			Bus:   client.SystemBus,
		},
	)
	return a
}

// ObjectManager manges the list of all available objects
type ObjectManager struct {
	client *client.Client
	logger *log.Logger
}

// Close the connection
func (o *ObjectManager) Close() {
	o.client.Disconnect()
}

// GetManagedObjects return a list of all available objects registered
func (o *ObjectManager) GetManagedObjects() (map[dbus.ObjectPath]map[string]map[string]dbus.Variant, error) {
	objects := make(map[dbus.ObjectPath]map[string]map[string]dbus.Variant)
	err := o.client.Call("GetManagedObjects", 0).Store(&objects)
	return objects, err
}

//Register watch for signal events
func (o *ObjectManager) Register() (chan *dbus.Signal, error) {
	path := o.client.Config.Path
	iface := o.client.Config.Iface
	return o.client.Register(path, iface)
}

//Unregister watch for signal events
func (o *ObjectManager) Unregister() error {
	path := o.client.Config.Path
	iface := o.client.Config.Iface
	return o.client.Unregister(path, iface)
}
package service_example

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	serviceAdapterID = "hci0"
	clientAdapterID  = "hci1"

	objectName      = "org.bluez"
	objectPath      = "/go_bluetooth/service"
	agentObjectPath = "/go_bluetooth/agent"
)

func fail(where string, err error) {
	if err != nil {
		log.Errorf("%s: %s", where, err)
		os.Exit(1)
	}
}

func Run() error {

	log.Warn("***\nThis example assume two controller are available: hci0 and hci1\n***")

	var err error

	log.Info("Register agent")
	agent, err := createAgent()
	fail("createAgent", err)

	defer agent.Release()

	log.Info("Register app")
	app, err := registerApplication(serviceAdapterID)
	fail("registerApplication", err)

	defer app.StopAdvertising()

	adapterProps, err := app.GetAdapter().GetProperties()
	fail("GetProperties", err)

	hwaddr := adapterProps.Address

	var serviceID string
	for _, service := range app.GetServices() {
		serviceID = service.GetProperties().UUID
		break
	}

	time.Sleep(time.Millisecond * 500)

	err = createClient(clientAdapterID, hwaddr, serviceID)
	fail("createClient", err)

	select {}
}
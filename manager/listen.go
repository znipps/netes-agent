package manager

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/event-subscriber/events"
)

func (m *Manager) Listen() error {
	logrus.Infof("Listening for events on %s", m.rancherClient.GetOpts().Url)

	router, err := events.NewEventRouter(m.rancherClient, 250, map[string]events.EventHandler{
		"instance.start":        wrapHandler(m.HandleComputeInstanceActivate),
		"deploymentunit.remove": wrapHandler(m.HandleComputeInstanceRemove),
		"cluster.remove":        wrapHandler(m.handleClusterRemove),
	})
	if err != nil {
		return err
	}

	return router.StartHandler("k8s-cluster-service", nil)
}

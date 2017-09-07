package sync

import (
	"testing"

	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/labels"
	"github.com/rancher/netes-agent/utils"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/pkg/api/v1"
)

func TestGetLabels(t *testing.T) {
	assert.Equal(t, getLabels(client.DeploymentSyncRequest{
		Revision:           "revision",
		DeploymentUnitUuid: "00000000-0000-0000-0000-000000000000",
	}), map[string]string{
		labels.RevisionLabel:       "revision",
		labels.DeploymentUuidLabel: "00000000-0000-0000-0000-000000000000",
	})
}

func TestGetAnnotations(t *testing.T) {
	assert.Equal(t, getAnnotations(client.DeploymentSyncRequest{
		Containers: []client.Container{
			{
				Name: "c1",
				Uuid: "00000000-0000-0000-0000-000000000000",
				Labels: map[string]interface{}{
					labels.ServiceLaunchConfig: labels.ServicePrimaryLaunchConfig,
					"a": "b",
				},
			},
			{
				Name: "c2",
				Uuid: "00000000-0000-0000-0000-000000000000",
				Labels: map[string]interface{}{
					"c": "d",
				},
			},
		},
	}), map[string]string{
		"a": "b",
		"c1/io.rancher.container.uuid": "00000000-0000-0000-0000-000000000000",
		"c2/c": "d",
		"c2/io.rancher.container.uuid": "00000000-0000-0000-0000-000000000000",
	})
}

func TestGetPodSpec(t *testing.T) {
	assert.Equal(t, getPodSpec(client.DeploymentSyncRequest{
		Containers: []client.Container{
			{},
		},
	}), v1.PodSpec{
		RestartPolicy: v1.RestartPolicyNever,
		DNSPolicy:     v1.DNSDefault,
	})

	assert.Equal(t, getPodSpec(client.DeploymentSyncRequest{
		Containers: []client.Container{
			{
				Labels: map[string]interface{}{
					labels.ServiceLaunchConfig: labels.ServicePrimaryLaunchConfig,
				},
				RestartPolicy: &client.RestartPolicy{
					Name: "always",
				},
				PrimaryNetworkId: "1",
				IpcMode:          "host",
				PidMode:          "host",
			},
		},
		Networks: []client.Network{
			{
				Resource: client.Resource{
					Id: "1",
				},
				Kind: hostNetworkingKind,
			},
		},
		NodeName: "node1",
	}), v1.PodSpec{
		RestartPolicy: v1.RestartPolicyNever,
		HostIPC:       true,
		HostNetwork:   true,
		HostPID:       true,
		DNSPolicy:     v1.DNSDefault,
		NodeName:      "node1",
	})
}

func TestSecurityContext(t *testing.T) {
	securityContext := getSecurityContext(client.Container{
		Privileged: true,
		ReadOnly:   true,
		CapAdd: []string{
			"capadd1",
			"capadd2",
		},
		CapDrop: []string{
			"capdrop1",
			"capdrop2",
		},
	})
	assert.Equal(t, *securityContext.Privileged, true)
	assert.Equal(t, *securityContext.ReadOnlyRootFilesystem, true)
	assert.Equal(t, securityContext.Capabilities.Add, []v1.Capability{
		v1.Capability("capadd1"),
		v1.Capability("capadd2"),
	})
	assert.Equal(t, securityContext.Capabilities.Drop, []v1.Capability{
		v1.Capability("capdrop1"),
		v1.Capability("capdrop2"),
	})
}

func TestGetHostAliases(t *testing.T) {
	assert.Equal(t, []v1.HostAlias{
		{
			IP: "0.0.0.0",
			Hostnames: []string{
				"hostname",
			},
		},
	}, getHostAliases(client.Container{
		ExtraHosts: []string{
			"hostname:0.0.0.0",
		},
	}))
}

func TestGetVolumes(t *testing.T) {
	assert.Equal(t, getVolumes(client.DeploymentSyncRequest{
		Containers: []client.Container{
			{
				DataVolumes: []string{
					"/host/path:/container/path",
				},
			},
		},
	}), []v1.Volume{
		{
			Name: utils.Hash("/host/path"),
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "/host/path",
				},
			},
		},
	})
	assert.Equal(t, getVolumes(client.DeploymentSyncRequest{
		Containers: []client.Container{
			{
				Tmpfs: map[string]interface{}{
					"/dir": true,
				},
			},
		},
	}), []v1.Volume{
		{
			Name: utils.Hash("/dir"),
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{
					Medium: v1.StorageMediumMemory,
				},
			},
		},
	})
	assert.Equal(t, len(getVolumes(client.DeploymentSyncRequest{
		Containers: []client.Container{
			{
				DataVolumes: []string{
					"/anonymous/volume",
				},
			},
		},
	})), 0)
}

func TestGetVolumeMounts(t *testing.T) {
	assert.Equal(t, getVolumeMounts(client.Container{
		DataVolumes: []string{
			"/host/path:/container/path",
		},
	}), []v1.VolumeMount{
		{
			Name:      utils.Hash("/host/path"),
			MountPath: "/container/path",
		},
	})
	assert.Equal(t, getVolumeMounts(client.Container{
		Tmpfs: map[string]interface{}{
			"/dir": true,
		},
	}), []v1.VolumeMount{
		{
			Name:      utils.Hash("/dir"),
			MountPath: "/dir",
		},
	})
	assert.Equal(t, len(getVolumeMounts(client.Container{
		DataVolumes: []string{
			"/anonymous/volume",
		},
	})), 0)
}

func TestGetAffinity(t *testing.T) {
	matchExpressions := getAffinity(client.Container{
		Labels: map[string]interface{}{
			labels.HostAffinityLabel:     "key1=val1,key2=val2",
			labels.HostAntiAffinityLabel: "key3=val3,key4=val4",
		},
	}).NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions
	assert.Len(t, matchExpressions, 4)
	for _, nodeSelectorRequirement := range []v1.NodeSelectorRequirement{
		{
			Key:      "key1",
			Operator: v1.NodeSelectorOpIn,
			Values: []string{
				"val1",
			},
		},
		{
			Key:      "key2",
			Operator: v1.NodeSelectorOpIn,
			Values: []string{
				"val2",
			},
		},
		{
			Key:      "key3",
			Operator: v1.NodeSelectorOpNotIn,
			Values: []string{
				"val3",
			},
		},
		{
			Key:      "key4",
			Operator: v1.NodeSelectorOpNotIn,
			Values: []string{
				"val4",
			},
		},
	} {
		assert.Contains(t, matchExpressions, nodeSelectorRequirement)
	}

	matchExpressions = getAffinity(client.Container{
		Labels: map[string]interface{}{
			labels.HostSoftAffinityLabel:     "key1=val1,key2=val2",
			labels.HostSoftAntiAffinityLabel: "key3=val3,key4=val4",
		},
	}).NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].Preference.MatchExpressions
	assert.Len(t, matchExpressions, 4)
	for _, nodeSelectorRequirement := range []v1.NodeSelectorRequirement{
		{
			Key:      "key1",
			Operator: v1.NodeSelectorOpIn,
			Values: []string{
				"val1",
			},
		},
		{
			Key:      "key2",
			Operator: v1.NodeSelectorOpIn,
			Values: []string{
				"val2",
			},
		},
		{
			Key:      "key3",
			Operator: v1.NodeSelectorOpNotIn,
			Values: []string{
				"val3",
			},
		},
		{
			Key:      "key4",
			Operator: v1.NodeSelectorOpNotIn,
			Values: []string{
				"val4",
			},
		},
	} {
		assert.Contains(t, matchExpressions, nodeSelectorRequirement)
	}
}

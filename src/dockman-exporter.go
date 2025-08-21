package main

import (
	"strings"
	"encoding/json"
	"io"
	"net"
    "net/http"
	"os"
	"log"
	"context"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

//Variables to store Docker/Podman API json
type Json map[string]interface{} //json{}
type DockerAPI []Json //json[{}]

var socket = findSocket()

//Prometheus specific struct
type dockerInfoCollector struct {
	dockerImageMetric *prometheus.Desc
	dockerContainerState *prometheus.Desc
}

//Connect to socket
func connectSocket(socket string) http.Client {
	return http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socket)
			},
		},
	}
}

//Find if docker or podman is used
func findSocket() string {
	var sock string
	sockets := [3]string{"/var/run/podman.sock", "/var/run/docker.sock", "/var/run/user/1000/podman/podman.sock"}
	
	for _, sock = range sockets {
		if _, err := os.Stat(sock); err == nil {
			log.Printf("Socket found: %s\n",sock)
			return sock
		} 
	}
	log.Fatal("No Podman or Docker socket found!")
	return sock 
}

//Connect to socket and trigger requests.
//To be expanded
func getContainerList(socket string) DockerAPI {
	uri := "/containers/json?all=true"

	httpc := connectSocket(socket)

	var response *http.Response
	var err error
	//List images
	response, err = httpc.Get("http://unix" + uri)

	if err != nil {
		log.Fatal(err)
	}
	var containersJson DockerAPI
	if response.StatusCode == 200 {
		
		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
		}

		
		if err := json.Unmarshal([]byte(body), &containersJson); err != nil {
        	log.Fatal(err)
    	}
		
	}
	return containersJson
}

func getImageList(socket string) DockerAPI {
	uri := "/images/json"

	httpc := connectSocket(socket)

	var response *http.Response
	var err error
	//List images
	response, err = httpc.Get("http://unix" + uri)

	if err != nil {
		log.Fatal(err)
	}
	var imagesJson DockerAPI
	if response.StatusCode == 200 {
		

		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
		}

		
		if err := json.Unmarshal([]byte(body), &imagesJson); err != nil {
        	log.Fatal(err)
    	}
		
	}
	return imagesJson
}

//Not really needed as can be labeled directly on prometheus
func getHostname(socket string) string {
	uri := "/info"

	httpc := connectSocket(socket)

	var response *http.Response
	var err error
	var hostname string
	//List images
	response, err = httpc.Get("http://unix" + uri)

	if err != nil {
		log.Fatal(err)
	}
	var infoJson Json
	if response.StatusCode == 200 {
		

		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
		}

		
		if err := json.Unmarshal([]byte(body), &infoJson); err != nil {
        	log.Fatal(err)
    	}
		hostname = infoJson["Name"].(string)
		
	}
	return hostname
}


//I barely understand how this works
func newDockerMetricsCollector() *dockerInfoCollector {
	return &dockerInfoCollector{
		dockerImageMetric: prometheus.NewDesc("docker_image_size", "Docker Image Size in bytes.",
			[]string{"hostname","docker_image_name", "docker_image_tag","docker_image_id"}, nil,
		),
		dockerContainerState: prometheus.NewDesc("docker_container_running_state", "Docker Running Container State. (-1=unknown,0=created,1=initialized,2=running,3=stopped,4=paused,5=exited,6=removing,7=stopping)",
			[]string{"container_name", "docker_image_name", "docker_image_id", "hostname"}, nil,
		),
	}
}

func (collector *dockerInfoCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.dockerImageMetric
	ch <- collector.dockerContainerState
}

func (collector *dockerInfoCollector) Collect(ch chan<- prometheus.Metric) {
	var imageName string
	var imageTag string
	var containerName string
	var containerState float64
	var hostname string
	imagesJson := getImageList(socket)
	containersJson := getContainerList(socket)
	hostname = getHostname(socket)
	
	for _, image := range imagesJson {
		if len(image["RepoTags"].([]interface{})) != 0 {
			img := image["RepoTags"].([]interface{})[0].(string)
			imageName = strings.Split(img, ":")[0]
			imageTag = strings.Split(img, ":")[1]

		} else if len(image["RepoDigests"].([]interface{})) != 0 {
			img := image["RepoDigests"].([]interface{})[0].(string)
			imageName = strings.Split(img, "@")[0]
			imageTag = "none"

		}
		imageId := image["Id"].(string)
		imageSize := image["Size"].(float64)
		m1 := prometheus.MustNewConstMetric(collector.dockerImageMetric, prometheus.GaugeValue, imageSize, hostname, imageName, imageTag, imageId )
		ch <- m1
	}

	for _, container := range containersJson {
		//status=(-1=unknown,0=created,1=initialized,2=running,3=stopped,4=paused,5=exited,6=removing,7=stopping)
		containerName = container["Names"].([]interface{})[0].(string)
		containerName = strings.Split(containerName, "/")[1]
		if container["State"].(string) == "unknown" {
			containerState = -1
		} else if container["State"].(string) == "created" {
			containerState = 0
		} else if container["State"].(string) == "initialized" {
			containerState = 1
		} else if container["State"].(string) == "running" {
			containerState = 2
		} else if container["State"].(string) == "stopped" {
			containerState = 3
		} else if container["State"].(string) == "paused" {
			containerState = 4
		} else if container["State"].(string) == "exited" {
			containerState = 5
		} else if container["State"].(string) == "removing" {
			containerState = 6
		} else if container["State"].(string) == "stopping" {
			containerState = 7
		}
		imageName = container["Image"].(string)
		imageId := container["ImageID"].(string)
		m2 := prometheus.MustNewConstMetric(collector.dockerContainerState, prometheus.GaugeValue, containerState, containerName, imageName, imageId, hostname )
		ch <- m2
	}

}

func main() {
	dockerMetrics := newDockerMetricsCollector()
	prometheus.MustRegister(dockerMetrics)

    http.Handle("/metrics", promhttp.Handler())
    http.ListenAndServe(":9910", nil)
	log.Fatal(http.ListenAndServe(":9910", nil))
}

package main

import (
	"log"

	"github.com/iot-go-sdk/pkg/config"
	"github.com/iot-go-sdk/pkg/dynreg"
)

func main() {
	cfg := config.NewConfig()

	cfg.Device.ProductKey = "iAAdekmy"
	cfg.Device.DeviceName = "gBpWrBwUXd"
	cfg.Device.ProductSecret = "7GOiAgSMi20N6Mtk"

	cfg.MQTT.Host = "iot.know-act.com"
	cfg.MQTT.Port = 80

	client := dynreg.NewHTTPDynRegClient(cfg)

	log.Println("Starting HTTP dynamic registration...")

	deviceSecret, err := client.Register()
	if err != nil {
		log.Fatalf("Dynamic registration failed: %v", err)
	}

	log.Printf("Dynamic registration successful!")
	log.Printf("Device Secret: %s", deviceSecret)

	cfg.Device.DeviceSecret = deviceSecret

	log.Println("You can now use the device secret for MQTT connection")
}

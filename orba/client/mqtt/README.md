## `orba/client/mqtt`

... is a package containing an MQTT client Orba will use to integrate with other systems (ie., Home Assistant)
for receiving events

### TODO

- [ ] Integrate logging from `paho.mqtt.golang` into orba's logrus setup
- [ ] What is needed in the `mqtt.Config` aside from host, port, username, password?
   - [ ] Connection testing from command line 
- [ ] How are events from MQTT going to create matching entities, entity types, etc?


### NOTES

https://pkg.go.dev/github.com/eclipse/paho.mqtt.golang#ClientOptions is used to instantiate a new mqtt client. Map the required arguments over into `mqtt.Config`.

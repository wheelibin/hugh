package constants

import "time"

const MainUpdateInterval = time.Minute
const MaxLightOverrideMinutes = 120

// bridge events
const EventBatchTypeUpdate = "update"

const EventTypeZigbeeConnectivity = "zigbee_connectivity"
const EventStatusConnectivityIssue = "connectivity_issue"
const EventStatusConnected = "connected"

const EventTypeLight = "light"

const HughUpdateWindow = 2 * time.Second
const OverrideToleranceBrightness = 1
const OverrideToleranceColourTemp = 5

const ChangeTypeBrightness = "brightness"
const ChangeTypeColourTemp = "colour temp"

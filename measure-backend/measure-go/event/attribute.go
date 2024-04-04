package event

import (
	"fmt"
	"measure-backend/measure-go/platform"

	"github.com/google/uuid"
)

// Attribute defines common attributes associated with each event.
type Attribute struct {
	// InstallationID is the unique identifier for an installation
	// of an app. Generated by the client.
	InstallationID uuid.UUID `json:"installation_id" binding:"required"`

	// AppVersion is the app's version identifier.
	AppVersion string `json:"app_version" binding:"required"`

	// AppBuild is the app's build identifier.
	AppBuild string `json:"app_build" binding:"required"`

	// AppUniqueID is the app's bundle identifier.
	AppUniqueID string `json:"app_unique_id" binding:"required"`

	// MeasureSDKVersion is the measure sdk version
	// identifier.
	MeasureSDKVersion string `json:"measure_sdk_version" binding:"required"`

	// Platform is the client's platform, like:
	// - android
	// - ios
	// - flutter
	Platform string `json:"platform" binding:"required"`

	// ThreadName is the thread on which the
	// event was captured.
	ThreadName string `json:"thread_name"`

	// UserID is the id of the app's end user.
	UserID string `json:"user_id"`

	// DeviceName is the name of the device.
	DeviceName string `json:"device_name"`

	// DeviceModel is the model of the device.
	DeviceModel string `json:"device_model"`

	// DeviceManufacturer is the name of the device's
	// manufacturer.
	DeviceManufacturer string `json:"device_manufacturer"`

	// DeviceType is the type of the device, like phone
	// or tablet.
	DeviceType string `json:"device_type"`

	// DeviceIsFoldable is true for foldable devices.
	DeviceIsFoldable bool `json:"device_is_foldable"`

	// DeviceIsPhysical is true for physical devices.
	DeviceIsPhysical bool `json:"device_is_physical"`

	// DeviceDensityDPI is the DPI density of the device.
	DeviceDensityDPI uint16 `json:"device_density_dpi"`

	// DeviceWidthPX is the screen width of the device
	// in pixels.
	DeviceWidthPX uint16 `json:"device_width_px"`

	// DeviceHeightPX is the screen height of the device
	// in pixels.
	DeviceHeightPX uint16 `json:"device_height_px"`

	// DeviceDensity is the density of the device.
	DeviceDensity float32 `json:"device_density"`

	// DeviceLocale is the rfc 5646 based locale
	// identifier.
	DeviceLocale string `json:"device_locale"`

	// OSName is the operating system's name
	OSName string `json:"os_name"`

	// OSVersion is the operating system's vesrion.
	OSVersion string `json:"os_version"`

	// NetworkType is the type of the network. One of
	// - wifi
	// - cellular
	// - vpn
	// - unknown
	// - no_network
	NetworkType string `json:"network_type"`

	// NetworkProvider is the wireless service provider.
	NetworkProvider string `json:"network_provider"`

	// NetworkGeneration is generation of the network.
	// One of
	// - 2g
	// - 3g
	// - 4g
	// - 5g
	NetworkGeneration string `json:"network_generation"`
}

// Validate validates an event's attributes.
func (a Attribute) Validate() error {
	const (
		maxThreadNameChars         = 64
		maxUserIDChars             = 128
		maxDeviceNameChars         = 32
		maxDeviceModelChars        = 32
		maxDeviceManufacturerChars = 32
		maxDeviceTypeChars         = 32
		maxOSNameChars             = 32
		maxOSVersionChars          = 32
		maxPlatformChars           = 32
		maxAppVersionChars         = 32
		maxAppBuildChars           = 32
		maxAppUniqueIDChars        = 128
		maxMeasureSDKVersion       = 16
		maxNetworkTypeChars        = 16
		maxNetworkGenerationChars  = 8
		maxNetworkProviderChars    = 64
		maxDeviceLocaleChars       = 64
	)

	if len(a.AppVersion) > maxAppVersionChars {
		return fmt.Errorf(`%q exceeds maximum allowed characters of %d`, `attributes.app_version`, maxAppVersionChars)
	}
	if len(a.AppBuild) > maxAppBuildChars {
		return fmt.Errorf(`%q exceeds maximum allowed characters of %d`, `attributes.app_build`, maxAppBuildChars)
	}
	if len(a.AppUniqueID) > maxAppUniqueIDChars {
		return fmt.Errorf(`%q exceeds maximum allowed characters of %d`, `attrubutes.app_unique_id`, maxAppUniqueIDChars)
	}
	if len(a.Platform) > maxPlatformChars {
		return fmt.Errorf(`%q exceeds maximum allowed characters of %d`, `attributes.platform`, maxPlatformChars)
	}
	if len(a.MeasureSDKVersion) > maxMeasureSDKVersion {
		return fmt.Errorf(`%q exceeds maximum allowed characters of %d`, `attributes.measure_sdk_version`, maxMeasureSDKVersion)
	}
	if a.Platform != platform.Android && a.Platform != platform.IOS {
		return fmt.Errorf(`%q does not contain a valid platform value`, `attributes.platform`)
	}
	if len(a.ThreadName) > maxThreadNameChars {
		return fmt.Errorf(`%q exceeds maximum allowed characters of %d`, `attributes.thread_name`, maxThreadNameChars)
	}
	if len(a.UserID) > maxUserIDChars {
		return fmt.Errorf(`%q exceeds maximum allowed characters of %d`, `attributes.user_id`, maxUserIDChars)
	}
	if len(a.DeviceName) > maxDeviceNameChars {
		return fmt.Errorf(`%q exceeds maximum allowed characters of %d`, `attributes.device_name`, maxDeviceNameChars)
	}
	if len(a.DeviceModel) > maxDeviceModelChars {
		return fmt.Errorf(`%q exceeds maximum allowed characters of %d`, `attributes.device_model`, maxDeviceModelChars)
	}
	if len(a.DeviceManufacturer) > maxDeviceManufacturerChars {
		return fmt.Errorf(`%q exceeds maximum allowed characters of %d`, `attributes.device_manufacturer`, maxDeviceManufacturerChars)
	}
	if len(a.DeviceType) > maxDeviceTypeChars {
		return fmt.Errorf(`%q exceeds maximum allowed characters of %d`, `attributes.device_type`, maxDeviceTypeChars)
	}
	if len(a.DeviceLocale) > maxDeviceLocaleChars {
		return fmt.Errorf(`%q exceeds maximum allowed characters of %d`, `attributes.device_locale`, maxDeviceLocaleChars)
	}
	if len(a.OSName) > maxOSNameChars {
		return fmt.Errorf(`%q exceeds maximum allowed characters of %d`, `attributes.os_name`, maxOSNameChars)
	}
	if len(a.OSVersion) > maxOSVersionChars {
		return fmt.Errorf(`%q exceeds maximum allowed characters of %d`, `attributes.os_version`, maxOSVersionChars)
	}
	if len(a.NetworkType) > maxNetworkTypeChars {
		return fmt.Errorf(`%q exceeds maximum allowed characters of %d`, `attributes.network_type`, maxNetworkTypeChars)
	}
	if len(a.NetworkGeneration) > maxNetworkGenerationChars {
		return fmt.Errorf(`%q exceeds maximum allowed characters of %d`, `attributes.network_generation`, maxNetworkGenerationChars)
	}
	if len(a.NetworkProvider) > maxNetworkProviderChars {
		return fmt.Errorf(`%q exceeds maximum allowed characters of %d`, `attributes.network_provider`, maxNetworkProviderChars)
	}

	return nil
}

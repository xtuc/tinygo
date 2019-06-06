// +build sam,atsamd51,itsybitsy_m4

package machine

import "device/sam"

// GPIO Pins
const (
	D0  = PA16 // UART0 RX
	D1  = PA17 // UART0 TX
	D2  = PA07
	D3  = PB22 // PWM available
	D4  = PA14 // PWM available
	D5  = PA15 // PWM available
	D6  = PB02 // PWM available
	D7  = PA18 // PWM available
	D8  = PB03 // PWM available
	D9  = PA19 // PWM available
	D10 = PA20 // can be used for PWM or UART1 TX
	D11 = PA21 // can be used for PWM or UART1 RX
	D12 = PA23 // PWM available
	D13 = PA22 // PWM available
)

// Analog pins
const (
	A0 = PA02 // ADC/AIN[0]
	A1 = PB05 // ADC/AIN[2]
	A2 = PB08 // ADC/AIN[3]
	A3 = PB09 // ADC/AIN[4]
	A4 = PA04 // ADC/AIN[5]
	A5 = PA06 // ADC/AIN[10]
)

const (
	LED = D13
)

// UART0 aka USBCDC pins
const (
	USBCDC_DM_PIN = PA24
	USBCDC_DP_PIN = PA25
)

// UART1 pins
const (
	UART_TX_PIN = D10
	UART_RX_PIN = D11
)

// I2C pins
const (
	SDA_PIN = PA12 // SDA: SERCOM3/PAD[0]
	SCL_PIN = PA13 // SCL: SERCOM3/PAD[1]
)

// I2C on the ItsyBitsy M4.
var (
	I2C0 = I2C{Bus: sam.SERCOM2_I2CM,
		SDA:     SDA_PIN,
		SCL:     SCL_PIN,
		PinMode: PinSERCOM}
)

// SPI pins
const (
	SPI0_SCK_PIN  = PA01 // SCK: SERCOM1/PAD[3]
	SPI0_MOSI_PIN = PA00 // MOSI: SERCOM1/PAD[1]
	SPI0_MISO_PIN = PB23 // MISO: SERCOM1/PAD[0]
)

// SPI on the ItsyBitsy M4.
var (
	SPI0 = SPI{Bus: sam.SERCOM1_SPI}
)

// I2S pins
const (
	I2S_SCK_PIN = PA10
	I2S_SD_PIN  = PA08
	I2S_WS_PIN  = NoPin // TODO: figure out what this is on ItsyBitsy M4.
)

// I2S on the ItsyBitsy M4.
// var (
// 	I2S0 = I2S{Bus: sam.I2S}
// )

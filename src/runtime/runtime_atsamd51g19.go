// +build sam,atsamd51,atsamd51g19

package runtime

import (
	"device/sam"
)

func initSERCOMClocks() {
	// Turn on clock to SERCOM0 for UART0
	sam.MCLK.APBAMASK.SetBits(sam.MCLK_APBAMASK_SERCOM0_)

	//GCLK->PCHCTRL[clk_core].reg = GCLK_PCHCTRL_GEN_GCLK1_Val | (1 << GCLK_PCHCTRL_CHEN_Pos);
  	//GCLK->PCHCTRL[clk_slow].reg = GCLK_PCHCTRL_GEN_GCLK3_Val | (1 << GCLK_PCHCTRL_CHEN_Pos);

	// Use GCLK0 for SERCOM0 aka UART0
	//GCLK->PCHCTRL[clk_id].reg =
	  //GCLK_PCHCTRL_GEN_GCLK1_Val | (1 << GCLK_PCHCTRL_CHEN_Pos);
	  
	// GCLK_CLKCTRL_ID( clockId ) | // Generic Clock 0 (SERCOMx)
	// GCLK_CLKCTRL_GEN_GCLK0 | // Generic Clock Generator 0 is source
	// GCLK_CLKCTRL_CLKEN ;
	sam.GCLK.PCHCTRL20.Set((sam.GCLK_PCHCTRL_GEN_GCLK1 << sam.GCLK_PCHCTRL_GEN_Pos) |
		sam.GCLK_PCHCTRL_CHEN)
	sam.GCLK.PCHCTRL19.Set((sam.GCLK_PCHCTRL_GEN_GCLK3 << sam.GCLK_PCHCTRL_GEN_Pos) |
		sam.GCLK_PCHCTRL_CHEN)
	//waitForSync()

	// Turn on clock to SERCOM1
	sam.MCLK.APBAMASK.SetBits(sam.MCLK_APBAMASK_SERCOM1_)
	sam.GCLK.PCHCTRL21.Set((sam.GCLK_PCHCTRL_GEN_GCLK1 << sam.GCLK_PCHCTRL_GEN_Pos) |
		sam.GCLK_PCHCTRL_CHEN)
	// sam.GCLK.PCHCTRL19.Set((sam.GCLK_PCHCTRL_GEN_GCLK3 << sam.GCLK_PCHCTRL_GEN_Pos) |
	// 	sam.GCLK_PCHCTRL_CHEN)

	// Turn on clock to SERCOM2
	sam.MCLK.APBBMASK.SetBits(sam.MCLK_APBBMASK_SERCOM2_)
	sam.GCLK.PCHCTRL22.Set((sam.GCLK_PCHCTRL_GEN_GCLK1 << sam.GCLK_PCHCTRL_GEN_Pos) |
		sam.GCLK_PCHCTRL_CHEN)

	// Turn on clock to SERCOM3
	sam.MCLK.APBBMASK.SetBits(sam.MCLK_APBBMASK_SERCOM3_)
	sam.GCLK.PCHCTRL23.Set((sam.GCLK_PCHCTRL_GEN_GCLK1 << sam.GCLK_PCHCTRL_GEN_Pos) |
		sam.GCLK_PCHCTRL_CHEN)

	// Turn on clock to SERCOM4
	sam.MCLK.APBDMASK.SetBits(sam.MCLK_APBDMASK_SERCOM4_)
	sam.GCLK.PCHCTRL24.Set((sam.GCLK_PCHCTRL_GEN_GCLK1 << sam.GCLK_PCHCTRL_GEN_Pos) |
		sam.GCLK_PCHCTRL_CHEN)


	// Turn on clock to SERCOM5
	sam.MCLK.APBDMASK.SetBits(sam.MCLK_APBDMASK_SERCOM5_)
	sam.GCLK.PCHCTRL25.Set((sam.GCLK_PCHCTRL_GEN_GCLK1 << sam.GCLK_PCHCTRL_GEN_Pos) |
		sam.GCLK_PCHCTRL_CHEN)
}

package command

import (
	"github.com/GMWalletApp/epusdt/config"
	"github.com/gookit/color"
)

func printBanner() {
	color.Green.Printf("%s\n", "  _____                     _ _   \n | ____|_ __  _   _ ___  __| | |_ \n |  _| | '_ \\| | | / __|/ _` | __|\n | |___| |_) | |_| \\__ \\ (_| | |_ \n |_____| .__/ \\__,_|___/\\__,_|\\__|\n       |_|                        ")
	color.Infof(
		"Epusdt version(%s) commit(%s) built(%s) Powered by %s %s \n",
		config.GetAppVersion(),
		config.GetBuildCommit(),
		config.GetBuildDate(),
		"GMwalletApp",
		"https://github.com/GMwalletApp/epusdt",
	)
}

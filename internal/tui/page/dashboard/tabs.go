package dashboard

// NextTab handles right/l navigation
func NextTab(current Tab) Tab {
	switch current {
	case TabOverview:
		return TabStrain // right from landing → rightmost (strain)
	case TabSleep:
		return TabRecovery
	case TabRecovery:
		return TabStrain
	case TabStrain:
		return TabSleep // wrap around
	default:
		return TabStrain
	}
}

// PrevTab handles left/h navigation
func PrevTab(current Tab) Tab {
	switch current {
	case TabOverview:
		return TabSleep // left from landing → leftmost (sleep)
	case TabSleep:
		return TabStrain // wrap around
	case TabRecovery:
		return TabSleep
	case TabStrain:
		return TabRecovery
	default:
		return TabSleep
	}
}

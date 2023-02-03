package strays

func (m *StrayManager) AddHand() {

	hand := LittleHand{
		Waiter: &m.Waiter,
		Stray:  nil,
	}

	m.hands = append(m.hands, hand)
}

func (m *StrayManager) Start(count int) {
	for i := 0; i < count; i++ {
		m.AddHand()
	}

	for {
		m.Waiter.Add(1)
		for i := 0; i < len(m.hands); i++ {
			if len(m.Strays) < i {
				continue
			}

			m.hands[i].Stray = m.Strays[i]
		}
	}
}

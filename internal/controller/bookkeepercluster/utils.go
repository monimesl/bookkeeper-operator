package bookkeepercluster

func mergeLabels(ms ...map[string]string) map[string]string {
	res := make(map[string]string)
	for _, m := range ms {
		for key, value := range m {
			res[key] = value
		}
	}
	return res
}

func mapEqual(m1, m2 map[string]string) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k, v := range m1 {
		if m2[k] != v {
			return false
		}
	}
	return true
}

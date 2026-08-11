package atlas

import "github.com/typed-web-commons/typed-web/internal/jsonbounded"

func jsonPolicy(maximum int) jsonbounded.Policy {
	return jsonbounded.Policy{MaxBytes: maximum, MaxDepth: 24, MaxScalarBytes: 16 << 10, MaxContainerEntries: 4096, MaxTokens: 500000}
}

func decodeStrict(data []byte, destination any, policy jsonbounded.Policy) error {
	return jsonbounded.Decode(data, destination, policy, true)
}

/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// Passbolt contains a secretRef for the passbolt credentials.
type PassboltAuth struct {
	PasswordSecretRef   *esmeta.SecretKeySelector `json:"passwordSecretRef"`
	PrivateKeySecretRef *esmeta.SecretKeySelector `json:"privateKeySecretRef"`
}

type PassboltProvider struct {
	// Auth defines the information necessary to authenticate against Passbolt Server
	Auth *PassboltAuth `json:"auth"`
	// Host defines the Passbolt Server to connect to
	Host string `json:"host"`
}

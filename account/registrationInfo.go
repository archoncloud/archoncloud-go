package account

import (
	"fmt"
	. "github.com/itsmeknt/archoncloud-go/common"
	"github.com/itsmeknt/archoncloud-go/interfaces"
	"github.com/pariz/gountries"
	"github.com/pkg/errors"
	"strings"
)

const RegistrationFileName = "registration.txt"
const RegistrationVersion = 2

func GetRegistrationInfo() (*interfaces.RegistrationInfo, error) {
	regPath := DefaultToExecutable(RegistrationFileName)
	r := interfaces.RegistrationInfo{
		"USA",
		1,
		interfaces.EthereumValues{3.5, 1.0},
		interfaces.NeoValues{0.002},
		RegistrationVersion,
	}
	if !FileExists(regPath) {
		// generate default and exit
		err := SaveConfiguration(&r, regPath)
		if err == nil {
			err = fmt.Errorf("You need to first fill in %s, then try again\n", regPath)
		} else {
			err = fmt.Errorf("Cannot save default registration template %s\n", regPath)
		}
		return nil, err
	}
	var errMsgs []string
	errP := fmt.Sprintf("in %s", regPath)
	err := GetConfiguration(&r, regPath)
	if err != nil {return nil, errors.WithMessage(err, errP)}

	if r.PledgedGigaBytes <= 0.0 {
		errMsgs = append(errMsgs,"pledged_giga_bytes must be larger than 0")
	}
	if r.Ethereum.EthStake <= 0.0 {
		errMsgs = append(errMsgs,"Eth_stake must be larger than 0")
	}
	if r.Ethereum.WeiPerByte <= 0.0 {
		errMsgs = append(errMsgs,"Wei_per_byte must be larger than 0")
	}
	if r.CountryA3 == "" {
		r.CountryA3 = "USA"
	} else {
		query := gountries.New()
		_, err := query.FindCountryByAlpha(r.CountryA3)
		if err != nil {
			errMsgs = append(errMsgs,"unknown country code " + r.CountryA3 + " (Must be ISO 3166-1 alpha-3)")
		}
	}
	if r.Version < RegistrationVersion {
		// New fields added
		SaveConfiguration(&r, regPath)
	}
	if len(errMsgs) == 0 {
		return &r, nil
	}
	return nil, errors.New(strings.Join(errMsgs, "\n\t"))
}

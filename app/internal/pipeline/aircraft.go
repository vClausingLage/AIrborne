package pipeline

// AircraftOption is one entry of the challenge-mode aircraft picker.
type AircraftOption struct {
	Type  string `json:"type"`  // aircraft/script name as the plan expects it
	Label string `json:"label"` // human readable
	Side  string `json:"side"`  // hint for the prompt: red | blue | any
}

// IL2Aircraft lists the flyable IL-2 Korea types (script names).
var IL2Aircraft = []AircraftOption{
	{Type: "mig15bis", Label: "MiG-15bis", Side: "red"},
	{Type: "il10", Label: "Il-10", Side: "red"},
	{Type: "la11", Label: "La-11", Side: "red"},
	{Type: "yak9p", Label: "Yak-9P", Side: "red"},
	{Type: "tu2", Label: "Tu-2", Side: "red"},
	{Type: "f86a5", Label: "F-86A-5 Sabre", Side: "blue"},
	{Type: "f80c10", Label: "F-80C-10 Shooting Star", Side: "blue"},
	{Type: "f84e", Label: "F-84E Thunderjet", Side: "blue"},
	{Type: "f51d", Label: "F-51D Mustang", Side: "blue"},
}

// DCSAircraft lists common flyable DCS modules by their mission "type" name.
// The picker also accepts free text for anything not listed.
var DCSAircraft = []AircraftOption{
	{Type: "F-16C_50", Label: "F-16C Viper", Side: "blue"},
	{Type: "FA-18C_hornet", Label: "F/A-18C Hornet", Side: "blue"},
	{Type: "F-15ESE", Label: "F-15E Strike Eagle", Side: "blue"},
	{Type: "F-15C", Label: "F-15C Eagle", Side: "blue"},
	{Type: "F-14B", Label: "F-14B Tomcat", Side: "blue"},
	{Type: "F-4E-45MC", Label: "F-4E Phantom II", Side: "blue"},
	{Type: "F-5E-3", Label: "F-5E Tiger II", Side: "any"},
	{Type: "A-10C_2", Label: "A-10C II Warthog", Side: "blue"},
	{Type: "AV8BNA", Label: "AV-8B Harrier II", Side: "blue"},
	{Type: "M-2000C", Label: "Mirage 2000C", Side: "blue"},
	{Type: "Mirage-F1CE", Label: "Mirage F1CE", Side: "any"},
	{Type: "AJS37", Label: "AJS-37 Viggen", Side: "blue"},
	{Type: "JF-17", Label: "JF-17 Thunder", Side: "red"},
	{Type: "MiG-21Bis", Label: "MiG-21bis", Side: "red"},
	{Type: "MiG-19P", Label: "MiG-19P", Side: "red"},
	{Type: "MiG-15bis", Label: "MiG-15bis", Side: "red"},
	{Type: "MiG-29S", Label: "MiG-29S", Side: "red"},
	{Type: "Su-27", Label: "Su-27 Flanker", Side: "red"},
	{Type: "Su-33", Label: "Su-33 Flanker-D", Side: "red"},
	{Type: "Su-25T", Label: "Su-25T Frogfoot", Side: "red"},
	{Type: "F-86F Sabre", Label: "F-86F Sabre", Side: "blue"},
	{Type: "L-39ZA", Label: "L-39ZA Albatros", Side: "red"},
	{Type: "C-101CC", Label: "C-101CC Aviojet", Side: "any"},
	{Type: "MB-339A", Label: "MB-339A", Side: "any"},
	{Type: "Ka-50_3", Label: "Ka-50 Black Shark III", Side: "red"},
	{Type: "Mi-24P", Label: "Mi-24P Hind", Side: "red"},
	{Type: "Mi-8MT", Label: "Mi-8MTV2 Hip", Side: "red"},
	{Type: "AH-64D_BLK_II", Label: "AH-64D Apache", Side: "blue"},
	{Type: "UH-1H", Label: "UH-1H Huey", Side: "blue"},
	{Type: "SA342M", Label: "SA342 Gazelle", Side: "blue"},
	{Type: "OH-58D", Label: "OH-58D Kiowa Warrior", Side: "blue"},
	{Type: "CH-47Fbl1", Label: "CH-47F Chinook", Side: "blue"},
	{Type: "P-51D-30-NA", Label: "P-51D Mustang", Side: "blue"},
	{Type: "P-47D-30", Label: "P-47D Thunderbolt", Side: "blue"},
	{Type: "SpitfireLFMkIX", Label: "Spitfire LF Mk.IX", Side: "blue"},
	{Type: "MosquitoFBMkVI", Label: "Mosquito FB Mk.VI", Side: "blue"},
	{Type: "Bf-109K-4", Label: "Bf 109 K-4", Side: "red"},
	{Type: "FW-190D9", Label: "Fw 190 D-9", Side: "red"},
	{Type: "FW-190A8", Label: "Fw 190 A-8", Side: "red"},
	{Type: "I-16", Label: "I-16", Side: "red"},
}

// DCSMaps are the theatres the DCS generator knows (see dcs.lookupTerrain).
var DCSMaps = []string{"Caucasus", "Syria", "PersianGulf", "Nevada", "Normandy", "TheChannel", "MarianaIslands", "Falklands", "Sinai", "Kola"}

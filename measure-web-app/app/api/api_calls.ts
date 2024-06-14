import { AppRouterInstance } from "next/dist/shared/lib/app-router-context.shared-runtime"
import { getAccessTokenOrRedirectToAuth, logoutIfAuthError } from "../utils/auth_utils"
import { supabase } from "@/utils/supabase/browser"
import { JourneyType } from "../components/journey"
import { UserInputDateType, formatUserInputDateToServerFormat } from "../utils/time_utils"

export enum TeamsApiStatus {
    Loading,
    Success,
    Error
}

export enum AppsApiStatus {
    Loading,
    Success,
    Error,
    NoApps
}

export enum FiltersApiStatus {
    Loading,
    Success,
    Error,
    NotOnboarded,
    NoData
}

export enum FiltersApiType {
    All,
    Crash,
    Anr
}

export enum JourneyApiStatus {
    Loading,
    Success,
    Error,
    NoData
}

export enum MetricsApiStatus {
    Loading,
    Success,
    Error
}

export enum CrashOrAnrType {
    Crash,
    Anr
}

export enum CrashOrAnrGroupsApiStatus {
    Loading,
    Success,
    Error
}

export enum ExceptionsOverviewPlotApiStatus {
    Loading,
    Success,
    Error,
    NoData
}

export enum CrashOrAnrGroupDetailsApiStatus {
    Loading,
    Success,
    Error
}

export enum CrashOrAnrGroupDetailsPlotApiStatus {
    Loading,
    Success,
    Error,
    NoData
}

export enum CreateTeamApiStatus {
    Init,
    Loading,
    Success,
    Error
}

export enum CreateAppApiStatus {
    Init,
    Loading,
    Success,
    Error
}

export enum TeamNameChangeApiStatus {
    Init,
    Loading,
    Success,
    Error
}

export enum RoleChangeApiStatus {
    Init,
    Loading,
    Success,
    Error
}

export enum InviteMemberApiStatus {
    Init,
    Loading,
    Success,
    Error
}

export enum RemoveMemberApiStatus {
    Init,
    Loading,
    Success,
    Error
}

export enum AuthzAndMembersApiStatus {
    Loading,
    Success,
    Error
}

export enum SessionReplayApiStatus {
    Loading,
    Success,
    Error
}

export enum FetchAlertPrefsApiStatus {
    Loading,
    Success,
    Error
}

export enum UpdateAlertPrefsApiStatus {
    Init,
    Loading,
    Success,
    Error
}

export const emptyTeam = { 'id': '', 'name': '' }

export const emptyApp = {
    "id": "",
    "team_id": "",
    "name": "",
    "api_key": {
        "created_at": "",
        "key": "",
        "last_seen": null,
        "revoked": false
    },
    "onboarded": false,
    "created_at": "",
    "updated_at": "",
    "platform": null,
    "onboarded_at": null,
    "unique_identifier": null
}

export const emptyJourney = {
    "links": [
        {
            "source": "",
            "target": "",
            "value": 0
        }
    ],
    "nodes": [
        {
            "id": "au.com.shiftyjelly.pocketcasts.ui.MainActivity",
            "issues": {
                "anrs": [
                    {
                        "id": "",
                        "title": "",
                        "count": 0
                    },
                ],
                "crashes": [
                    {
                        "id": "",
                        "title": "",
                        "count": 0
                    },
                ]
            }
        }
    ],
    "totalIssues": 0
}

export const emptyMetrics = {
    "adoption": {
        "all_versions": 0,
        "selected_version": 0,
        "adoption": 0,
        "nan": false
    },
    "anr_free_sessions": {
        "anr_free_sessions": 0,
        "delta": 0,
        "nan": false
    },
    "cold_launch": {
        "delta": 0,
        "nan": false,
        "p95": 0
    },
    "crash_free_sessions": {
        "crash_free_sessions": 0,
        "delta": 0,
        "nan": false
    },
    "hot_launch": {
        "delta": 0,
        "nan": false,
        "p95": 0
    },
    "perceived_anr_free_sessions": {
        "perceived_anr_free_sessions": 0,
        "delta": 0,
        "nan": false
    },
    "perceived_crash_free_sessions": {
        "perceived_crash_free_sessions": 0,
        "delta": 0,
        "nan": false
    },
    "sizes": {
        "average_app_size": 0,
        "selected_app_size": 0,
        "delta": 0,
        "nan": false
    },
    "warm_launch": {
        "delta": 0,
        "nan": false,
        "p95": 0
    }
}

const emptyCrashOrAnrGroup = {
    "id": "",
    "app_id": "",
    "name": "",
    "fingerprint": "",
    "count": 0,
    "percentage_contribution": 0,
    "created_at": "",
    "updated_at": ""
}

export const emptyCrashOrAnrGroupsResponse = {
    "meta": {
        "next": false,
        "previous": false
    },
    "results": [] as typeof emptyCrashOrAnrGroup[]
}

const emptyCrashGroupDetails = {
    "id": "",
    "session_id": "",
    "timestamp": "",
    "type": "",
    "thread_name": "",
    "attribute": {
        "installation_id": "",
        "app_version": "",
        "app_build": "",
        "app_unique_id": "",
        "measure_sdk_version": "",
        "platform": "",
        "thread_name": "",
        "user_id": "",
        "device_name": "",
        "device_model": "",
        "device_manufacturer": "",
        "device_type": "",
        "device_is_foldable": false,
        "device_is_physical": false,
        "device_density_dpi": 0,
        "device_width_px": 0,
        "device_height_px": 0,
        "device_density": 0.0,
        "device_locale": "",
        "os_name": "",
        "os_version": "",
        "network_type": "",
        "network_provider": "",
        "network_generation": ""
    },
    "exception": {
        "title": "",
        "stacktrace": ""
    },
    "attachments": [
        {
            "id": "",
            "name": "",
            "type": "",
            "key": "",
            "location": ""
        }
    ],
    "threads": [
        {
            "name": "",
            "frames": [
                ""
            ]
        }
    ],
    "attributes": {}
}

export const emptyCrashGroupDetailsResponse = {
    "meta": {
        "next": true,
        "previous": false
    },
    "results": [] as typeof emptyCrashGroupDetails[]
}

const emptyAnrGroupDetails = {
    "id": "",
    "session_id": "",
    "timestamp": "",
    "type": "",
    "thread_name": "",
    "attribute": {
        "installation_id": "",
        "app_version": "",
        "app_build": "",
        "app_unique_id": "",
        "measure_sdk_version": "",
        "platform": "",
        "thread_name": "",
        "user_id": "",
        "device_name": "",
        "device_model": "",
        "device_manufacturer": "",
        "device_type": "",
        "device_is_foldable": false,
        "device_is_physical": false,
        "device_density_dpi": 0,
        "device_width_px": 0,
        "device_height_px": 0,
        "device_density": 0.0,
        "device_locale": "",
        "os_name": "",
        "os_version": "",
        "network_type": "",
        "network_provider": "",
        "network_generation": ""
    },
    "anr": {
        "title": "",
        "stacktrace": ""
    },
    "attachments": [
        {
            "id": "",
            "name": "",
            "type": "",
            "key": "",
            "location": ""
        }
    ],
    "threads": [
        {
            "name": "",
            "frames": [
                ""
            ]
        }
    ],
    "attributes": {}
}

export const emptyAnrGroupDetailsResponse = {
    "meta": {
        "next": true,
        "previous": false
    },
    "results": [] as typeof emptyAnrGroupDetails[]
}

export const emptyExceptionsOverviewPlotResponse = [
    {
        "id": "",
        "data": [
            {
                "crash_free_sessions": 0,
                "datetime": "",
                "instances": 0
            }
        ]
    }
]

export const emptyCrashOrAnrGroupDetailsPlotResponse = [
    {
        "id": "",
        "data": [
            {
                "datetime": "",
                "instances": 0
            }
        ]
    }
]

export const defaultAuthzAndMembers = {
    "can_invite": [
        "viewer"
    ],
    "members": [
        {
            "id": "",
            "name": null,
            "email": "",
            "role": "",
            "last_sign_in_at": "",
            "created_at": "",
            "authz": {
                "can_change_roles": [
                    ""
                ],
                "can_remove": true
            }
        }
    ]
}

export const emptySessionReplay = {
    "app_id": "2b7ddad4-40a6-42a7-9e21-a90577e08263",
    "attribute": {
        "installation_id": "",
        "app_version": "",
        "app_build": "",
        "app_unique_id": "",
        "measure_sdk_version": "",
        "platform": "",
        "thread_name": "",
        "user_id": "",
        "device_name": "",
        "device_model": "",
        "device_manufacturer": "",
        "device_type": "",
        "device_is_foldable": true,
        "device_is_physical": false,
        "device_density_dpi": 0,
        "device_width_px": 0,
        "device_height_px": 0,
        "device_density": 0.0,
        "device_locale": "",
        "os_name": "",
        "os_version": "",
        "network_type": "",
        "network_provider": "",
        "network_generation": ""
    },
    "cpu_usage": [
        {
            "timestamp": "",
            "value": 0.0
        }
    ],
    "duration": 0,
    "memory_usage": [
        {
            "java_max_heap": 0,
            "java_total_heap": 0,
            "java_free_heap": 0,
            "total_pss": 0,
            "rss": 0,
            "native_total_heap": 0,
            "native_free_heap": 0,
            "interval_config": 0,
            "timestamp": ""
        }
    ],
    "session_id": "",
    "threads": {
        "main": [
            {
                "event_type": "lifecycle_activity",
                "thread_name": "",
                "type": "",
                "class_name": "",
                "intent": "",
                "saved_instance_state": false,
                "timestamp": ""
            },
            {
                "event_type": "lifecycle_app",
                "thread_name": "",
                "type": "",
                "timestamp": ""
            },
            {
                "event_type": "exception",
                "title": "",
                "thread_name": "",
                "handled": false,
                "stacktrace": "",
                "foreground": true,
                "timestamp": "",
                "attachments": [
                    {
                        "id": "",
                        "name": "",
                        "type": "",
                        "key": "",
                        "location": ""
                    }
                ]
            }
        ]
    }
}

export const emptyAlertPrefs = {
    crash_rate_spike: {
        email: true
    },
    anr_rate_spike: {
        email: true
    },
    launch_time_spike: {
        email: true
    }
}

export class AppVersion {
    name: string;
    code: string;
    displayName: string;

    constructor(name: string, code: string) {
        this.name = name;
        this.code = code;
        this.displayName = this.name + ' (' + this.code + ')'
    }
}

export const fetchTeamsFromServer = async (router: AppRouterInstance) => {
    const authToken = await getAccessTokenOrRedirectToAuth(supabase, router)
    const origin = process.env.NEXT_PUBLIC_API_BASE_URL
    const opts = {
        headers: {
            "Authorization": `Bearer ${authToken}`
        }
    };

    const res = await fetch(`${origin}/teams`, opts);
    if (!res.ok) {
        logoutIfAuthError(supabase, router, res)
        return { status: TeamsApiStatus.Error, data: null }
    }

    const data: [{ id: string, name: string }] = await res.json()

    return { status: TeamsApiStatus.Success, data: data }
}

export const fetchAppsFromServer = async (teamId: string, router: AppRouterInstance) => {
    const authToken = await getAccessTokenOrRedirectToAuth(supabase, router)
    const origin = process.env.NEXT_PUBLIC_API_BASE_URL
    const opts = {
        headers: {
            "Authorization": `Bearer ${authToken}`
        }
    };

    const res = await fetch(`${origin}/teams/${teamId}/apps`, opts);

    if (!res.ok && res.status == 404) {
        return { status: AppsApiStatus.NoApps, data: null }
    }

    if (!res.ok) {
        logoutIfAuthError(supabase, router, res)
        return { status: AppsApiStatus.Error, data: null }
    }

    const data = await res.json()
    return { status: AppsApiStatus.Success, data: data }
}

export const fetchFiltersFromServer = async (selectedApp: typeof emptyApp, filtersApiType: FiltersApiType, router: AppRouterInstance) => {
    if (!selectedApp.onboarded) {
        return { status: FiltersApiStatus.NotOnboarded, data: null }
    }

    const authToken = await getAccessTokenOrRedirectToAuth(supabase, router)
    const origin = process.env.NEXT_PUBLIC_API_BASE_URL
    const opts = {
        headers: {
            "Authorization": `Bearer ${authToken}`
        }
    };

    let url = `${origin}/apps/${selectedApp.id}/filters`

    // if filter is for Crashes or Anrs, we append a query param indicating it
    if (filtersApiType === FiltersApiType.Crash) {
        url += '?crash=1'
    } else if (filtersApiType === FiltersApiType.Anr) {
        url += '?anr=1'
    }

    const res = await fetch(url, opts);

    if (!res.ok) {
        logoutIfAuthError(supabase, router, res)
        return { status: FiltersApiStatus.Error, data: null }
    }

    const data = await res.json()

    if (data.versions === null) {
        return { status: FiltersApiStatus.NoData, data: null }
    }

    return { status: FiltersApiStatus.Success, data: data }
}

export const fetchJourneyFromServer = async (appId: string, journeyType: JourneyType, crashOrAnrGroupId: string | null, bidirectional: boolean, startDate: string, endDate: string, appVersions: AppVersion[], countries: string[], networkProviders: string[], networkTypes: string[], networkGenerations: string[], locales: string[], deviceManufacturers: string[], deviceNames: string[], router: AppRouterInstance) => {
    // Must pass in crashOrAnrGroupdId if journey type is crash or anr details
    if ((journeyType === JourneyType.CrashDetails || journeyType === JourneyType.AnrDetails) && crashOrAnrGroupId === undefined) {
        return { status: JourneyApiStatus.Error, data: null }
    }

    const authToken = await getAccessTokenOrRedirectToAuth(supabase, router)
    const origin = process.env.NEXT_PUBLIC_API_BASE_URL
    const opts = {
        headers: {
            "Authorization": `Bearer ${authToken}`
        }
    };

    let url = ''
    if (journeyType === JourneyType.CrashDetails) {
        url = `${origin}/apps/${appId}/crashGroups/${crashOrAnrGroupId}/plots/journey`
    } else if (journeyType === JourneyType.AnrDetails) {
        url = `${origin}/apps/${appId}/anrGroups/${crashOrAnrGroupId}/plots/journey`
    } else {
        url = `${origin}/apps/${appId}/journey`
    }

    // Append dates
    const serverFormattedStartDate = formatUserInputDateToServerFormat(startDate, UserInputDateType.From)
    const serverFormattedEndDate = formatUserInputDateToServerFormat(endDate, UserInputDateType.To)
    url = url + `?from=${serverFormattedStartDate}&to=${serverFormattedEndDate}`

    // Append versions if present
    if (appVersions.length > 0) {
        url = url + `&versions=${Array.from(appVersions).map((v) => v.name).join(',')}`
        url = url + `&version_codes=${Array.from(appVersions).map((v) => v.code).join(',')}`
    }

    // Append bidirectional value
    url = url + `&bigraph=${bidirectional ? '1' : '0'}`

    // Append countries if present
    if (countries.length > 0) {
        url = url + `&countries=${Array.from(countries).join(',')}`
    }

    // Append network providers if present
    if (networkProviders.length > 0) {
        url = url + `&network_providers=${Array.from(networkProviders).join(',')}`
    }

    // Append network types if present
    if (networkTypes.length > 0) {
        url = url + `&network_types=${Array.from(networkTypes).join(',')}`
    }

    // Append network generations if present
    if (networkGenerations.length > 0) {
        url = url + `&network_generations=${Array.from(networkGenerations).join(',')}`
    }

    // Append locales if present
    if (locales.length > 0) {
        url = url + `&locales=${Array.from(locales).join(',')}`
    }

    // Append device manufacturers if present
    if (deviceManufacturers.length > 0) {
        url = url + `&device_manufacturers=${Array.from(deviceManufacturers).join(',')}`
    }

    // Append device names if present
    if (deviceNames.length > 0) {
        url = url + `&device_names=${Array.from(deviceNames).join(',')}`
    }

    const res = await fetch(url, opts);

    if (!res.ok) {
        logoutIfAuthError(supabase, router, res)
        return { status: JourneyApiStatus.Error, data: null }
    }

    const data = await res.json()

    return { status: JourneyApiStatus.Success, data: data }
}

export const fetchMetricsFromServer = async (appId: string, startDate: string, endDate: string, appVersions: AppVersion[], router: AppRouterInstance) => {
    const authToken = await getAccessTokenOrRedirectToAuth(supabase, router)
    const origin = process.env.NEXT_PUBLIC_API_BASE_URL
    const opts = {
        headers: {
            "Authorization": `Bearer ${authToken}`
        }
    };

    const serverFormattedStartDate = formatUserInputDateToServerFormat(startDate, UserInputDateType.From)
    const serverFormattedEndDate = formatUserInputDateToServerFormat(endDate, UserInputDateType.To)

    let url = `${origin}/apps/${appId}/metrics?from=${serverFormattedStartDate}&to=${serverFormattedEndDate}`

    // Append versions if present
    if (appVersions.length > 0) {
        url = url + `&versions=${Array.from(appVersions).map((v) => v.name).join(',')}`
        url = url + `&version_codes=${Array.from(appVersions).map((v) => v.code).join(',')}`
    }

    const res = await fetch(url, opts);

    if (!res.ok) {
        logoutIfAuthError(supabase, router, res)
        return { status: MetricsApiStatus.Error, data: null }
    }

    const data = await res.json()

    return { status: MetricsApiStatus.Success, data: data }
}

export const fetchCrashOrAnrGroupsFromServer = async (crashOrAnrType: CrashOrAnrType, appId: string, startDate: string, endDate: string, appVersions: AppVersion[], keyId: string | null, limit: number, router: AppRouterInstance) => {
    const authToken = await getAccessTokenOrRedirectToAuth(supabase, router)
    const origin = process.env.NEXT_PUBLIC_API_BASE_URL
    const opts = {
        headers: {
            "Authorization": `Bearer ${authToken}`
        }
    };

    const serverFormattedStartDate = formatUserInputDateToServerFormat(startDate, UserInputDateType.From)
    const serverFormattedEndDate = formatUserInputDateToServerFormat(endDate, UserInputDateType.To)

    var url = ""
    if (crashOrAnrType === CrashOrAnrType.Crash) {
        url = `${origin}/apps/${appId}/crashGroups?from=${serverFormattedStartDate}&to=${serverFormattedEndDate}&limit=${limit}`
    } else {
        url = `${origin}/apps/${appId}/anrGroups?from=${serverFormattedStartDate}&to=${serverFormattedEndDate}&limit=${limit}`
    }

    // Append versions if present
    if (appVersions.length > 0) {
        url = url + `&versions=${Array.from(appVersions).map((v) => v.name).join(',')}`
        url = url + `&version_codes=${Array.from(appVersions).map((v) => v.code).join(',')}`
    }

    // Append keyId if present
    if (keyId !== null) {
        url = url + `&key_id=${keyId}`
    }

    const res = await fetch(url, opts);

    if (!res.ok) {
        logoutIfAuthError(supabase, router, res)
        return { status: CrashOrAnrGroupsApiStatus.Error, data: null }
    }

    const data = await res.json()

    return { status: CrashOrAnrGroupsApiStatus.Success, data: data }

}

export const fetchCrashOrAnrGroupDetailsFromServer = async (crashOrAnrType: CrashOrAnrType, appId: string, crashOrAnrGroupId: string, startDate: string, endDate: string, appVersions: AppVersion[], countries: string[], networkProviders: string[], networkTypes: string[], networkGenerations: string[], locales: string[], deviceManufacturers: string[], deviceNames: string[], keyId: string | null, keyTimestamp: string | null, limit: number, router: AppRouterInstance) => {
    const authToken = await getAccessTokenOrRedirectToAuth(supabase, router)
    const origin = process.env.NEXT_PUBLIC_API_BASE_URL
    const opts = {
        headers: {
            "Authorization": `Bearer ${authToken}`
        }
    };

    const serverFormattedStartDate = formatUserInputDateToServerFormat(startDate, UserInputDateType.From)
    const serverFormattedEndDate = formatUserInputDateToServerFormat(endDate, UserInputDateType.To)

    var url = ""
    if (crashOrAnrType === CrashOrAnrType.Crash) {
        url = `${origin}/apps/${appId}/crashGroups/${crashOrAnrGroupId}/crashes?from=${serverFormattedStartDate}&to=${serverFormattedEndDate}&limit=${limit}`
    } else {
        url = `${origin}/apps/${appId}/anrGroups/${crashOrAnrGroupId}/anrs?from=${serverFormattedStartDate}&to=${serverFormattedEndDate}&limit=${limit}`
    }

    // Append versions if present
    if (appVersions.length > 0) {
        url = url + `&versions=${Array.from(appVersions).map((v) => v.name).join(',')}`
        url = url + `&version_codes=${Array.from(appVersions).map((v) => v.code).join(',')}`
    }

    // Append countries if present
    if (countries.length > 0) {
        url = url + `&countries=${Array.from(countries).join(',')}`
    }

    // Append network providers if present
    if (networkProviders.length > 0) {
        url = url + `&network_providers=${Array.from(networkProviders).join(',')}`
    }

    // Append network types if present
    if (networkTypes.length > 0) {
        url = url + `&network_types=${Array.from(networkTypes).join(',')}`
    }

    // Append network generations if present
    if (networkGenerations.length > 0) {
        url = url + `&network_generations=${Array.from(networkGenerations).join(',')}`
    }

    // Append locales if present
    if (locales.length > 0) {
        url = url + `&locales=${Array.from(locales).join(',')}`
    }

    // Append device manufacturers if present
    if (deviceManufacturers.length > 0) {
        url = url + `&device_manufacturers=${Array.from(deviceManufacturers).join(',')}`
    }

    // Append device names if present
    if (deviceNames.length > 0) {
        url = url + `&device_names=${Array.from(deviceNames).join(',')}`
    }

    // Append keyId if present
    if (keyId !== null) {
        url = url + `&key_id=${keyId}`
    }

    // Append keyTimestamp if present
    if (keyTimestamp !== null) {
        url = url + `&key_timestamp=${keyTimestamp}`
    }

    const res = await fetch(url, opts);

    if (!res.ok) {
        logoutIfAuthError(supabase, router, res)
        return { status: CrashOrAnrGroupDetailsApiStatus.Error, data: null }
    }

    const data = await res.json()

    return { status: CrashOrAnrGroupDetailsApiStatus.Success, data: data }

}

export const fetchExceptionsOverviewPlotFromServer = async (appId: string, crashOrAnrType: CrashOrAnrType, startDate: string, endDate: string, appVersions: AppVersion[], router: AppRouterInstance) => {
    const authToken = await getAccessTokenOrRedirectToAuth(supabase, router)
    const origin = process.env.NEXT_PUBLIC_API_BASE_URL
    const opts = {
        headers: {
            "Authorization": `Bearer ${authToken}`
        }
    };

    const serverFormattedStartDate = formatUserInputDateToServerFormat(startDate, UserInputDateType.From)
    const serverFormattedEndDate = formatUserInputDateToServerFormat(endDate, UserInputDateType.To)

    var url = ""
    if (crashOrAnrType === CrashOrAnrType.Crash) {
        url = `${origin}/apps/${appId}/crashGroups/plots/instances?from=${serverFormattedStartDate}&to=${serverFormattedEndDate}`
    } else {
        url = `${origin}/apps/${appId}/anrGroups/plots/instances?from=${serverFormattedStartDate}&to=${serverFormattedEndDate}`
    }

    // Append versions if present
    if (appVersions.length > 0) {
        url = url + `&versions=${Array.from(appVersions).map((v) => v.name).join(',')}`
        url = url + `&version_codes=${Array.from(appVersions).map((v) => v.code).join(',')}`
    }

    const res = await fetch(url, opts);

    if (!res.ok) {
        logoutIfAuthError(supabase, router, res)
        return { status: ExceptionsOverviewPlotApiStatus.Error, data: null }
    }

    const data = await res.json()

    if (data === null) {
        return { status: ExceptionsOverviewPlotApiStatus.NoData, data: null }
    }

    return { status: ExceptionsOverviewPlotApiStatus.Success, data: data }
}


export const fetchCrashOrAnrGroupDetailsPlotFromServer = async (appId: string, crashOrAnrType: CrashOrAnrType, crashOrAnrGroupId: string, startDate: string, endDate: string, appVersions: AppVersion[], countries: string[], networkProviders: string[], networkTypes: string[], networkGenerations: string[], locales: string[], deviceManufacturers: string[], deviceNames: string[], router: AppRouterInstance) => {
    const authToken = await getAccessTokenOrRedirectToAuth(supabase, router)
    const origin = process.env.NEXT_PUBLIC_API_BASE_URL
    const opts = {
        headers: {
            "Authorization": `Bearer ${authToken}`
        }
    };

    const serverFormattedStartDate = formatUserInputDateToServerFormat(startDate, UserInputDateType.From)
    const serverFormattedEndDate = formatUserInputDateToServerFormat(endDate, UserInputDateType.To)

    var url = ""
    if (crashOrAnrType === CrashOrAnrType.Crash) {
        url = `${origin}/apps/${appId}/crashGroups/${crashOrAnrGroupId}/plots/instances?from=${serverFormattedStartDate}&to=${serverFormattedEndDate}`
    } else {
        url = `${origin}/apps/${appId}/anrGroups/${crashOrAnrGroupId}/plots/instances?from=${serverFormattedStartDate}&to=${serverFormattedEndDate}`
    }

    // Append versions if present
    if (appVersions.length > 0) {
        url = url + `&versions=${Array.from(appVersions).map((v) => v.name).join(',')}`
        url = url + `&version_codes=${Array.from(appVersions).map((v) => v.code).join(',')}`
    }

    // Append countries if present
    if (countries.length > 0) {
        url = url + `&countries=${Array.from(countries).join(',')}`
    }

    // Append network providers if present
    if (networkProviders.length > 0) {
        url = url + `&network_providers=${Array.from(networkProviders).join(',')}`
    }

    // Append network types if present
    if (networkTypes.length > 0) {
        url = url + `&network_types=${Array.from(networkTypes).join(',')}`
    }

    // Append network generations if present
    if (networkGenerations.length > 0) {
        url = url + `&network_generations=${Array.from(networkGenerations).join(',')}`
    }

    // Append locales if present
    if (locales.length > 0) {
        url = url + `&locales=${Array.from(locales).join(',')}`
    }

    // Append device manufacturers if present
    if (deviceManufacturers.length > 0) {
        url = url + `&device_manufacturers=${Array.from(deviceManufacturers).join(',')}`
    }

    // Append device names if present
    if (deviceNames.length > 0) {
        url = url + `&device_names=${Array.from(deviceNames).join(',')}`
    }

    const res = await fetch(url, opts);

    if (!res.ok) {
        logoutIfAuthError(supabase, router, res)
        return { status: CrashOrAnrGroupDetailsPlotApiStatus.Error, data: null }
    }

    const data = await res.json()

    if (data === null) {
        return { status: CrashOrAnrGroupDetailsPlotApiStatus.NoData, data: null }
    }

    return { status: CrashOrAnrGroupDetailsPlotApiStatus.Success, data: data }
}

export const fetchAuthzAndMembersFromServer = async (teamId: string, router: AppRouterInstance) => {
    const authToken = await getAccessTokenOrRedirectToAuth(supabase, router)
    const origin = process.env.NEXT_PUBLIC_API_BASE_URL
    const opts = {
        headers: {
            "Authorization": `Bearer ${authToken}`
        }
    };

    const res = await fetch(`${origin}/teams/${teamId}/authz`, opts);
    if (!res.ok) {
        logoutIfAuthError(supabase, router, res)
        return { status: AuthzAndMembersApiStatus.Error, data: null }
    }

    const data = await res.json()

    return { status: AuthzAndMembersApiStatus.Success, data: data }
}

export const fetchSessionReplayFromServer = async (appId: string, sessionId: string, router: AppRouterInstance) => {
    const authToken = await getAccessTokenOrRedirectToAuth(supabase, router)
    const origin = process.env.NEXT_PUBLIC_API_BASE_URL
    const opts = {
        headers: {
            "Authorization": `Bearer ${authToken}`
        }
    };

    const res = await fetch(`${origin}/apps/${appId}/sessions/${sessionId}`, opts);
    if (!res.ok) {
        logoutIfAuthError(supabase, router, res)
        return { status: SessionReplayApiStatus.Error, data: null }
    }

    const data = await res.json()

    return { status: SessionReplayApiStatus.Success, data: data }
}

export const changeTeamNameFromServer = async (teamId: string, newTeamName: string, router: AppRouterInstance) => {
    const authToken = await getAccessTokenOrRedirectToAuth(supabase, router)
    const origin = process.env.NEXT_PUBLIC_API_BASE_URL
    const opts = {
        method: 'PATCH',
        headers: {
            "Authorization": `Bearer ${authToken}`
        },
        body: JSON.stringify({ name: newTeamName })
    };

    const res = await fetch(`${origin}/teams/${teamId}/rename`, opts);
    if (!res.ok) {
        logoutIfAuthError(supabase, router, res)
        return { status: TeamNameChangeApiStatus.Error }
    }

    return { status: TeamNameChangeApiStatus.Success }
}

export const createTeamFromServer = async (teamName: string, router: AppRouterInstance) => {
    const authToken = await getAccessTokenOrRedirectToAuth(supabase, router)
    const origin = process.env.NEXT_PUBLIC_API_BASE_URL
    const opts = {
        method: 'POST',
        headers: {
            "Authorization": `Bearer ${authToken}`
        },
        body: JSON.stringify({ name: teamName })
    };

    const res = await fetch(`${origin}/teams`, opts);
    const data = await res.json()

    if (!res.ok) {
        logoutIfAuthError(supabase, router, res)
        return { status: CreateTeamApiStatus.Error, error: data.error }
    }

    return { status: CreateTeamApiStatus.Success }
}

export const createAppFromServer = async (teamId: string, appName: string, router: AppRouterInstance) => {
    const authToken = await getAccessTokenOrRedirectToAuth(supabase, router)
    const origin = process.env.NEXT_PUBLIC_API_BASE_URL
    const opts = {
        method: 'POST',
        headers: {
            "Authorization": `Bearer ${authToken}`
        },
        body: JSON.stringify({ name: appName })
    };

    const res = await fetch(`${origin}/teams/${teamId}/apps`, opts);
    const data = await res.json()

    if (!res.ok) {
        logoutIfAuthError(supabase, router, res)
        return { status: CreateAppApiStatus.Error, error: data.error }
    }

    return { status: CreateAppApiStatus.Success, data: data }
}

export const changeRoleFromServer = async (teamId: string, newRole: string, memberId: string, router: AppRouterInstance) => {
    const authToken = await getAccessTokenOrRedirectToAuth(supabase, router)
    const origin = process.env.NEXT_PUBLIC_API_BASE_URL
    const opts = {
        method: 'PATCH',
        headers: {
            "Authorization": `Bearer ${authToken}`
        },
        body: JSON.stringify({ role: newRole.toLocaleLowerCase() })
    };

    const res = await fetch(`${origin}/teams/${teamId}/members/${memberId}/role`, opts);
    const data = await res.json()

    if (!res.ok) {
        logoutIfAuthError(supabase, router, res)
        return { status: RoleChangeApiStatus.Error, error: data.error }
    }

    return { status: RoleChangeApiStatus.Success }
}

export const inviteMemberFromServer = async (teamId: string, email: string, role: string, router: AppRouterInstance) => {
    const lowerCaseRole = role.toLocaleLowerCase()
    const opts = {
        method: 'POST',
        headers: {
            "Content-Type": `application/json`,
        },
        body: JSON.stringify({ teamId: teamId, email: email, role: lowerCaseRole })
    };

    const res = await fetch(`/auth/invite`, opts);
    const data = await res.json()

    if (!res.ok) {
        logoutIfAuthError(supabase, router, res)
        return { status: InviteMemberApiStatus.Error, error: data.error }
    }

    return { status: InviteMemberApiStatus.Success }
}

export const removeMemberFromServer = async (teamId: string, memberId: string, router: AppRouterInstance) => {
    const authToken = await getAccessTokenOrRedirectToAuth(supabase, router)
    const origin = process.env.NEXT_PUBLIC_API_BASE_URL
    const opts = {
        method: 'DELETE',
        headers: {
            "Authorization": `Bearer ${authToken}`
        },
    };

    const res = await fetch(`${origin}/teams/${teamId}/members/${memberId}`, opts);
    const data = await res.json()

    if (!res.ok) {
        logoutIfAuthError(supabase, router, res)
        return { status: RemoveMemberApiStatus.Error, error: data.error }
    }

    return { status: RemoveMemberApiStatus.Success }
}

export const fetchAlertPrefsFromServer = async (appId: string, router: AppRouterInstance) => {
    const authToken = await getAccessTokenOrRedirectToAuth(supabase, router)
    const origin = process.env.NEXT_PUBLIC_API_BASE_URL
    const opts = {
        headers: {
            "Authorization": `Bearer ${authToken}`
        }
    };

    const res = await fetch(`${origin}/apps/${appId}/alertPrefs`, opts);

    if (!res.ok) {
        logoutIfAuthError(supabase, router, res)
        return { status: FetchAlertPrefsApiStatus.Error, data: null }
    }

    const data = await res.json()

    return { status: FetchAlertPrefsApiStatus.Success, data: data }
}

export const updateAlertPrefsFromServer = async (appdId: string, alertPrefs: typeof emptyAlertPrefs, router: AppRouterInstance) => {
    const authToken = await getAccessTokenOrRedirectToAuth(supabase, router)
    const origin = process.env.NEXT_PUBLIC_API_BASE_URL
    const opts = {
        method: 'PATCH',
        headers: {
            "Authorization": `Bearer ${authToken}`
        },
        body: JSON.stringify(alertPrefs)
    };

    const res = await fetch(`${origin}/apps/${appdId}/alertPrefs`, opts);
    const data = await res.json()

    if (!res.ok) {
        logoutIfAuthError(supabase, router, res)
        return { status: UpdateAlertPrefsApiStatus.Error, error: data.error }
    }

    return { status: UpdateAlertPrefsApiStatus.Success }
}
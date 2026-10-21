import java.util.Properties

plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
    id("org.jetbrains.kotlin.plugin.compose")
}

android {
    namespace = "id.gmimsiloam.jemaat"
    compileSdk = 35

    defaultConfig {
        applicationId = "id.gmimsiloam.jemaat"
        minSdk = 26
        targetSdk = 35
        // the one place for the version: `make android-publish VERSION=x.y.z`
        // sets it; versionCode follows from it (1.2.3 → 10203), so it always
        // grows with the version, as Android requires for updates
        versionName = "1.3.0"
        val parts = versionName!!.split(".").mapNotNull { it.toIntOrNull() }
        require(parts.size == 3 && parts[1] < 100 && parts[2] < 100) {
            "versionName must be like 1.2.0 (major.minor.patch; minor and patch up to 99), not \"$versionName\""
        }
        versionCode = parts[0] * 10_000 + parts[1] * 100 + parts[2]
    }

    // The release key, from android/keystore.properties (never committed; see
    // keystore.properties.example). Without it the release APK is unsigned.
    val keystore = rootProject.file("keystore.properties").takeIf { it.exists() }?.let { f ->
        Properties().apply { f.inputStream().use { load(it) } }.also { p ->
            val missing = listOf("storeFile", "storePassword", "keyAlias", "keyPassword").filter { p.getProperty(it).isNullOrBlank() }
            require(missing.isEmpty()) { "android/keystore.properties is missing ${missing.joinToString()} (see keystore.properties.example)" }
            require(rootProject.file(p.getProperty("storeFile")).exists()) {
                "the key file ${p.getProperty("storeFile")} in android/keystore.properties does not exist"
            }
        }
    }
    signingConfigs {
        if (keystore != null) {
            create("release") {
                storeFile = rootProject.file(keystore.getProperty("storeFile"))
                storePassword = keystore.getProperty("storePassword")
                keyAlias = keystore.getProperty("keyAlias")
                keyPassword = keystore.getProperty("keyPassword")
            }
        }
    }

    buildTypes {
        release {
            if (keystore != null) signingConfig = signingConfigs.getByName("release")
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
        }
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    kotlinOptions {
        jvmTarget = "17"
    }
    buildFeatures {
        compose = true
        buildConfig = true
    }
    testOptions {
        unitTests.isReturnDefaultValues = true
    }
}

dependencies {
    val composeBom = platform("androidx.compose:compose-bom:2024.12.01")
    implementation(composeBom)
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.material:material-icons-core")
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.ui:ui-tooling-preview")
    debugImplementation("androidx.compose.ui:ui-tooling")

    implementation("androidx.core:core-ktx:1.15.0")
    implementation("androidx.activity:activity-compose:1.9.3")
    implementation("androidx.lifecycle:lifecycle-runtime-compose:2.8.7")
    implementation("androidx.lifecycle:lifecycle-viewmodel-compose:2.8.7")
    implementation("androidx.work:work-runtime-ktx:2.10.0")

    testImplementation("junit:junit:4.13.2")
    testImplementation("org.json:json:20240303") // real org.json for JVM unit tests
}

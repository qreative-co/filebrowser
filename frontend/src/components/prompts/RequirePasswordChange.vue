<template>
  <div class="card-content">
    <div v-if="error !== ''" class="wrong-login card">{{ error }}</div>
    <p>{{ $t("login.requirePasswordChangeInstructions") }}</p>
    <!-- Qreative fork: in-session mode (opened from Layout.vue) asks for the current password too. -->
    <template v-if="askCurrentPassword">
      <label for="current-password">{{ $t("login.requirePasswordChangeCurrent") }}</label>
      <input
        id="current-password"
        ref="currentPasswordInput"
        class="input"
        :class="{ 'form-invalid': error !== '' }"
        type="password"
        autocomplete="current-password"
        v-model="currentPassword"
      />
    </template>
    <label for="new-password">{{ $t("general.password") }}</label>
    <input
      id="new-password"
      ref="newPasswordInput"
      class="input"
      :class="{ 'form-invalid': showFieldInvalid }"
      type="password"
      autocomplete="new-password"
      v-model="newPassword"
      :placeholder="$t('settings.enterPassword')"
    />
    <input
      class="input"
      :class="{ 'form-invalid': showFieldInvalid }"
      type="password"
      autocomplete="new-password"
      v-model="passwordConfirm"
      :placeholder="$t('settings.enterPasswordAgain')"
      @keydown.enter.prevent="submit"
    />
    <PasswordRequirementsHint :password="newPassword" :confirm-password="passwordConfirm" />
    <template v-if="needsOtpInput">
      <label for="require-change-otp">{{ $t("otp.codeInputPlaceholder") }}</label>
      <input
        id="require-change-otp"
        class="input"
        :class="{ 'form-invalid': showFieldInvalid }"
        type="text"
        autocomplete="one-time-code"
        v-model="otp"
        @keydown.enter.prevent="submit"
      />
    </template>
  </div>

  <div class="card-actions">
    <button
      v-if="askCurrentPassword"
      type="button"
      class="button button--flat button--grey"
      :disabled="submitInFlight"
      @click="signOut"
    >
      {{ $t("general.logout", { suffix: "" }) }}
    </button>
    <button
      type="button"
      class="button button--flat button--blue"
      :disabled="submitInFlight || !canSubmit"
      @click="submit"
    >
      {{ $t("general.update") }}
    </button>
  </div>
</template>

<script>
import { mutations } from "@/store";
import { notify } from "@/notify";
import { authApi } from "@/api";
import { initAuth, logout } from "@/utils/auth";
import { evaluatePasswordPolicy } from "@/utils/passwordPolicy.js";
import PasswordRequirementsHint from "@/components/PasswordRequirementsHint.vue";

export default {
  name: "requirePasswordChange",
  components: {
    PasswordRequirementsHint,
  },
  props: {
    redirect: {
      type: String,
      default: "",
    },
    username: {
      type: String,
      default: "",
    },
    password: {
      type: String,
      default: "",
    },
    /** When true, show the TOTP field immediately (user already uses 2FA). */
    otpRequired: {
      type: Boolean,
      default: false,
    },
    /** Prompt id (passed by Prompts.vue); used to close a locked prompt. */
    promptId: {
      type: Number,
      default: undefined,
    },
  },
  data() {
    return {
      error: "",
      newPassword: "",
      passwordConfirm: "",
      otp: "",
      submitInFlight: false,
      showOtpField: false,
      currentPassword: this.password,
      // Opened in-session (no password from the login form): ask for the current one.
      askCurrentPassword: String(this.password ?? "") === "",
    };
  },
  computed: {
    needsOtpInput() {
      return this.otpRequired || this.showOtpField;
    },
    passwordPolicy() {
      return evaluatePasswordPolicy(this.newPassword, this.passwordConfirm);
    },
    showFieldInvalid() {
      if (this.error !== "") {
        return true;
      }
      const p = this.passwordPolicy;
      return p.showMismatch || (p.showLengthHint && !p.minLengthMet);
    },
    canSubmit() {
      if (
        String(this.currentPassword ?? "").trim() === "" ||
        !this.passwordPolicy.valid ||
        this.newPassword === this.currentPassword
      ) {
        return false;
      }
      if (this.needsOtpInput && String(this.otp ?? "").trim() === "") {
        return false;
      }
      return true;
    },
  },
  mounted() {
    // Focus the first field (same delay as the global v-focus directive).
    setTimeout(() => {
      const el = this.askCurrentPassword ? this.$refs.currentPasswordInput : this.$refs.newPasswordInput;
      el?.focus?.();
    }, 100);
  },
  watch: {
    currentPassword() {
      this.error = "";
    },
    newPassword() {
      this.error = "";
    },
    passwordConfirm() {
      this.error = "";
    },
  },
  methods: {
    async signOut() {
      await logout();
    },
    otpRequiredForMessage(message) {
      const text = String(message ?? "");
      return text.includes("OTP code is required for user");
    },
    async submit(event) {
      event?.preventDefault?.();
      if (this.submitInFlight || !this.canSubmit) {
        return;
      }
      if (!this.passwordPolicy.valid) {
        this.error = this.$t("settings.passwordRequirementsMinLength", {
          min: this.passwordPolicy.minLength,
        });
        notify.showError(this.error);
        return;
      }
      if (this.newPassword === this.currentPassword) {
        this.error = this.$t("login.requirePasswordChangeMustDiffer");
        notify.showError(this.error);
        return;
      }
      this.submitInFlight = true;
      try {
        await authApi.changeRequiredPassword(this.username, this.currentPassword, this.newPassword, this.otp);
        await initAuth();
        notify.showSuccessToast(this.$t("login.requirePasswordChangeSuccess"));
        // force: the in-session prompt is locked (not dismissable) until this succeeds.
        mutations.closeTopPrompt(this.promptId, { force: true });
        if (this.askCurrentPassword) {
          // Reload so every view re-fetches with the now unrestricted session.
          window.location.reload();
          return;
        }
        const path = this.redirect !== "" ? this.redirect : "/files/";
        await this.$router.push(path);
      } catch (err) {
        const message = err.message || this.$t("login.failedLogin");
        if (this.otpRequiredForMessage(message)) {
          this.showOtpField = true;
        }
        this.error = message;
        notify.showError(this.error);
      } finally {
        this.submitInFlight = false;
      }
    },
  },
};
</script>

<style scoped>
.wrong-login {
  background: var(--red) !important;
  color: #fff;
  padding: 0.5em;
  text-align: center;
  margin-bottom: 0.5em;
}

.input {
  margin-bottom: 0.5em;
}
</style>

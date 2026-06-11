<template>
  <div>
    <q-list style="max-width: 600px" dense>
      <q-item>
        <q-item-section>
          <q-item-label>自动转换字幕文件编码</q-item-label>
          <q-item-label caption>自动转换到目标编码，如果不是特殊情况，不建议开启，仅对新下载字幕生效</q-item-label>
          <q-item v-if="form.auto_change_sub_encode.enable">
            <q-item-section avatar top>
              <q-radio
                v-for="(v, k) in DESC_ENCODE_TYPE_NAME_MAP"
                :key="k"
                :label="v"
                v-model="form.auto_change_sub_encode.des_encode_type"
                :val="~~k"
              />
            </q-item-section>
          </q-item>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle v-model="form.auto_change_sub_encode.enable" />
        </q-item-section>
      </q-item>

      <q-separator spaced inset></q-separator>

      <q-item tag="label" :disable="!isChsChtChangerEnable" v-ripple>
        <q-item-section>
          <q-item-label>简、繁字幕互转功能</q-item-label>
          <q-item-label caption
            >需要开启"自动转换字幕文件编码"功能，并设置为转码"UTF-8"，否则无法启用和生效</q-item-label
          >
          <q-item v-if="form.chs_cht_changer.enable">
            <q-item-section avatar top>
              <q-radio
                :disable="!isChsChtChangerEnable"
                v-for="(v, k) in AUTO_CONVERT_LANG_NAME_MAP"
                :key="k"
                :label="v"
                v-model="form.chs_cht_changer.des_chinese_language_type"
                :val="~~k"
              />
            </q-item-section>
          </q-item>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle :disable="!isChsChtChangerEnable" v-model="form.chs_cht_changer.enable" />
        </q-item-section>
      </q-item>

      <q-separator spaced inset></q-separator>

      <q-item>
        <q-item-section>
          <q-item-label>远程Chrome</q-item-label>
          <q-item-label caption>
            本功能用于把依赖 Chrome
            的实验性抓取流程移到一台算力和资源更充足的机器上。当前默认字幕链路不依赖它，只有在明确启用相关实验性方案时才需要配置。<br />
            需要自行参看<a
              class="text-primary"
              href="https://go-rod.github.io/i18n/zh-CN/#/custom-launch?id=远程管理启动器"
              target="_blank"
              >https://go-rod.github.io/i18n/zh-CN/#/custom-launch?id=远程管理启动器</a
            >文档部署。该功能仍属实验性质，可用性和稳定性都没有保证，后续也可能随 go-rod 变化而调整。
          </q-item-label>
        </q-item-section>
        <q-item-section avatar>
          <q-toggle v-model="form.remote_chrome_settings.enable" />
        </q-item-section>
      </q-item>

      <template v-if="form.remote_chrome_settings.enable">
        <q-item>
          <q-item-section>
            <q-item-label>Remote Chrome ws URL</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input
              v-model="form.remote_chrome_settings.remote_docker_url"
              placeholder="ws://192.168.xx.xx:9222"
              standout
              dense
              :rules="[(val) => (form.remote_chrome_settings.enable && !!val) || '不能为空']"
            />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>Remote user data dir</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input
              v-model="form.remote_chrome_settings.remote_user_data_dir"
              placeholder="/mnt/share/tmp"
              standout
              dense
            />
          </q-item-section>
        </q-item>
      </template>

      <q-separator spaced inset />

      <q-item>
        <q-item-section>
          <q-item-label>本地Chrome</q-item-label>
          <q-item-label caption>
            开启后程序会优先使用内置自动探测到的本机 Chrome，可执行文件路径不再需要手动填写。
          </q-item-label>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle v-model="form.local_chrome_settings.enabled" />
        </q-item-section>
      </q-item>

      <template v-if="form.local_chrome_settings.enabled">
        <q-item>
          <q-item-section>
            <q-item-label caption>当前模式下将自动探测本机 Chrome，无需额外配置路径。</q-item-label>
          </q-item-section>
        </q-item>
      </template>

      <q-separator spaced inset />

      <q-item>
        <q-item-section>
          <q-item-label>英文字幕翻译保底</q-item-label>
          <q-item-label caption>
            只在中文字幕阶段失败后触发，改走英文字幕源并调用 LLM 翻译。当前仅对单字幕保存模式生效。
          </q-item-label>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle v-model="form.llm_subtitle_fallback.enable" />
        </q-item-section>
      </q-item>

      <template v-if="form.llm_subtitle_fallback.enable">
        <q-item>
          <q-item-section>
            <q-item-label>Provider</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input v-model="form.llm_subtitle_fallback.provider" standout dense />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>Base URL</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input
              v-model="form.llm_subtitle_fallback.base_url"
              placeholder="OpenAI-compatible base URL，可留空走 Gemini 原生"
              standout
              dense
            />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>API key</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input
              v-model="form.llm_subtitle_fallback.api_key"
              type="password"
              placeholder="留空则回退到 subflow 本地配置或环境变量"
              standout
              dense
            />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>Model</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input v-model="form.llm_subtitle_fallback.model" standout dense />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>Python executable</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input
              v-model="form.llm_subtitle_fallback.python_executable"
              placeholder="留空使用环境默认 python"
              standout
              dense
            />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>Subflow root</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input v-model="form.llm_subtitle_fallback.subflow_root_dir" standout dense />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>Log dir</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input v-model="form.llm_subtitle_fallback.log_dir" standout dense />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>Source language</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input v-model="form.llm_subtitle_fallback.source_language" standout dense />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>Target language</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input v-model="form.llm_subtitle_fallback.target_language" standout dense />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>Translate style</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input v-model="form.llm_subtitle_fallback.translate_style" placeholder="可留空" standout dense />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>仅在无中文候选时触发</q-item-label>
            <q-item-label caption>当前运行时就是这个语义，这里保留显式开关。</q-item-label>
          </q-item-section>
          <q-item-section avatar top>
            <q-toggle v-model="form.llm_subtitle_fallback.only_when_no_chinese_candidate" />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>保留英文源字幕副本</q-item-label>
          </q-item-section>
          <q-item-section avatar top>
            <q-toggle v-model="form.llm_subtitle_fallback.keep_english_source_copy" />
          </q-item-section>
        </q-item>
      </template>

      <q-separator spaced inset />

      <q-item>
        <q-item-section>
          <q-item-label>API key</q-item-label>
          <q-item-label caption>
            本程序提供了一些面向开发者的接口，通过 API key 鉴权。具体参见
            <!-- eslint-disable -->
            <a
              href="https://github.com/ChineseSubFinder/ChineseSubFinder/blob/docs/DesignFile/ApiKey%E8%AE%BE%E8%AE%A1/ApiKey%E8%AE%BE%E8%AE%A1.md"
              class="text-primary"
              target="_blank"
              >开发文档</a
            >
          </q-item-label>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle v-model="form.api_key_settings.enabled" />
        </q-item-section>
      </q-item>

      <template v-if="form.api_key_settings.enabled">
        <q-item>
          <q-btn label="重新生成密钥" color="primary" size="sm" @click="generateApiKey" />
        </q-item>
        <q-item class="q-mt-sm">
          <q-item-section>
            <q-input
              v-model="form.api_key_settings.key"
              standout
              dense
              :rules="[(val) => !!val || '不能为空']"
              readonly
            >
              <template #append>
                <copy-to-clipboard-btn v-if="form.api_key_settings.key" :text="form.api_key_settings.key" size="sm" />
              </template>
            </q-input>
          </q-item-section>
        </q-item>
      </template>
    </q-list>
  </div>
</template>

<script setup>
import { formModel } from 'pages/settings/use-settings';
import { toRefs } from '@vueuse/core';
import {
  AUTO_CONVERT_LANG_NAME_MAP,
  DESC_ENCODE_TYPE_NAME_MAP,
  DESC_ENCODE_TYPE_UTF8,
} from 'src/constants/SettingConstants';
import { computed } from 'vue';
import CopyToClipboardBtn from 'components/CopyToClipboardBtn';

const { experimental_function: form } = toRefs(formModel);

const isChsChtChangerEnable = computed(
  () =>
    formModel.experimental_function.auto_change_sub_encode?.enable &&
    formModel.experimental_function.auto_change_sub_encode?.des_encode_type === DESC_ENCODE_TYPE_UTF8
);

const generateUuid = () =>
  'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    // eslint-disable-next-line no-bitwise
    const r = (Math.random() * 16) | 0;
    // eslint-disable-next-line no-bitwise
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });

const generateApiKey = () => {
  const uuid = generateUuid();
  formModel.experimental_function.api_key_settings.key = uuid;
};
</script>

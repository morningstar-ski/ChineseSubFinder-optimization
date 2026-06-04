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
            <q-item-label>远程 Docker 地址</q-item-label>
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
            <q-item-label>远程 Docker 中的 ADBlocker 目录</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input
              v-model="form.remote_chrome_settings.remote_adblock_path"
              placeholder="/mnt/share/adblock1"
              standout
              dense
              :rules="[(val) => (form.remote_chrome_settings.enable && !!val) || '不能为空']"
            />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>远程 Docker 中的缓存文件夹目录</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input
              v-model="form.remote_chrome_settings.remote_user_data_dir"
              placeholder="/mnt/share/tmp"
              standout
              dense
              :rules="[(val) => (form.remote_chrome_settings.enable && !!val) || '不能为空']"
            />
          </q-item-section>
        </q-item>
      </template>

      <q-separator spaced inset />

      <q-item>
        <q-item-section>
          <q-item-label>本地Chrome</q-item-label>
          <q-item-label caption>
            只有在你明确启用了依赖 Chrome 的实验性抓取链路时，才需要填写这里。如果本程序能够自动下载 Chrome，
            就不建议手动指定版本；程序升级后，自动下载的 Chrome 会一起更新，而手动指定的版本需要你自己维护。下载动作由
            go-rod 负责，遇到兼容性问题通常也需要回到 go-rod 侧排查。注意以下几点：
            <div>
              <ol>
                <li>
                  如果是 Docker 用户，推荐把你解压后的 /volume1/docker/chinesesubfinder/Chrome 文件夹映射到
                  /app/cache/Plugin/Chrome 文件夹中，那么这里填写的就应该是 Chrome
                  在容器内的完整路径（举例，按你自己下载的 Chrome 来改）: /app/cache/Plugin/Chrome/chrome
                </li>
                <li>如果是 Windows 用户，那么就是你 Chrome.exe 的完整路径</li>
                <li>Chrome 版本不要太低</li>
                <li>请确认指定的 Chrome 与目标平台、CPU 架构一致</li>
              </ol>
            </div>
          </q-item-label>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle v-model="form.local_chrome_settings.enabled" />
        </q-item-section>
      </q-item>

      <template v-if="form.local_chrome_settings.enabled">
        <q-item>
          <q-item-section>
            <q-input
              v-model="form.local_chrome_settings.local_chrome_exe_f_path"
              label="Chrome(.exe) 的完整路径"
              placeholder="/your/chrome/path/chrome.exe"
              standout
              dense
              :rules="[(val) => !!val || '不能为空']"
            />
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

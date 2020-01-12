{ config, pkgs, lib, ... }:

let
  hostname = "changeme"; # e.g. donate.dumpstack.io
  github_token = "changeme"; # https://github.com/settings/tokens/new, no any scopes required
  ssh_key = "changeme"; # e.g. ssh-rsa AAA.....== user@localhost

  donate_src = fetchGit { url = "https://code.dumpstack.io/tools/donate"; };
  donate = import "${donate_src}";
  database_path = "/home/donate/donate.db.sqlite3";
in {
  imports =
    [ # Include the results of the hardware scan.
      ./hardware-configuration.nix
    ];

  boot.loader.grub.enable = true;
  boot.loader.grub.version = 2;
  boot.loader.grub.device = "/dev/vda";

  networking.hostName = "${hostname}";

  time.timeZone = "UTC";

  services.openssh.enable = true;

  networking.firewall =  {
    enable = true;
    allowedTCPPorts = [ 80 443 ];
  };

  swapDevices = [
    { device = "/var/swapfile";
      size = 2048; # MiB
    }
  ];

  users.extraUsers.root = {
    openssh.authorizedKeys.keys = [ ssh_key ];
  };

  services.nginx = {
    enable = true;
    virtualHosts."${hostname}" = {
      enableACME = true;
      forceSSL = true;

      locations."/".proxyPass = "http://127.0.0.1:8080";
    };
  };

  users.users.donate = {
    isNormalUser = true;
  };

  systemd.services."donate" = {
    serviceConfig = {
      User = "donate";
      ExecStart = "${donate}/bin/donate --database ${database_path} --token ${github_token}";
      Restart = "on-failure";
    };
    wantedBy = [ "default.target" ];
  };

  system.stateVersion = "19.09";
  system.autoUpgrade.enable = true;

  nix = {
    optimise.automatic = true;
    gc = {
      automatic = true;
      options = "--delete-older-than 7d";
    };
  };
}

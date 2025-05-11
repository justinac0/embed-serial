use std::time::Duration;

#[derive(serde::Deserialize, serde::Serialize)]
#[serde(default)]
pub struct SerialEmbedApp {
    #[serde(skip)]
    current_port: Option<Box<dyn serialport::SerialPort + 'static>>,
    ports: Vec<String>,
    console: Vec<String>,
    message: String,
}

impl Default for SerialEmbedApp {
    fn default() -> Self {
        Self {
            current_port: None,
            ports: Vec::new(),
            console: Vec::new(),
            message: String::new(),
        }
    }
}

impl SerialEmbedApp {
    pub fn new(_: &eframe::CreationContext<'_>) -> Self {
        Default::default()
    }
}

impl eframe::App for SerialEmbedApp {
    fn update(&mut self, ctx: &egui::Context, _frame: &mut eframe::Frame) {

        egui::CentralPanel::default().show(ctx, |ui| {
            let available_width = ui.available_width();
            let left_width = available_width / 3.0;
            let right_width = available_width - left_width;

            ui.set_width(available_width);
            ui.heading("SerialEmbed");

            ui.horizontal(|ui| {
                ui.label("Send Message: ");
                ui.text_edit_singleline(&mut self.message);
                if ui.button("Send").clicked() {
                    if self.current_port.is_some() {
                        let msg = self.message.as_bytes(); 
                        let port = self.current_port.as_mut().unwrap();
                        let result = port.write(msg);
                        if result.is_ok() {
                            println!("{:?}", result);
                        }
                    }
                }
            });

            ui.separator();

            ui.horizontal(|ui| {
                ui.set_width(available_width);
                // config panel
                if self.current_port.is_none() {
                    ui.vertical(|ui| {
                        if ui.button("Scan COM Ports").clicked() {
                            ui.set_width(left_width);
                            
                            let ports = serialport::available_ports().unwrap();
                            self.ports.clear();
                            for p in ports {
                                self.ports.push(p.port_name);
                            }
                        }
                        
                    for p in &self.ports {
                        let name = p.as_str();

                        ui.horizontal(|ui| {
                        ui.label(name);
                            if ui.button("Open").clicked() {
                                self.current_port = Some(serialport::new(name, 115_200)
                                    .timeout(Duration::from_millis(100))
                                    .open()
                                    .expect("Failed to open port"));
                            }
                        });
                    }
                });
            }

                // content panel
                ui.vertical(|ui| {
                    ui.set_width(right_width);
                    ui.label("Right panel (2/3)");
                });
            });

            // footer
            ui.with_layout(egui::Layout::bottom_up(egui::Align::RIGHT), |ui| {
                ui.hyperlink("https://justinchappell.com");
            });
        });
    }
}
